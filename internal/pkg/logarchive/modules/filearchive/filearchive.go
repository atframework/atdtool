package filearchive

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/atframework/atdtool/internal/pkg/logarchive"
	"github.com/atframework/atdtool/internal/pkg/logarchive/modules/cos"
	"github.com/fsnotify/fsnotify"
	"github.com/shirou/gopsutil/v3/disk"
	"go.uber.org/zap"
)

type fileStatus int

const (
	fileStatusUnKnown fileStatus = iota
	fileStatusWaitUpload
	fileStatusUploading
	fileStatusUploaded
)

type notifyType int

const (
	notifyTypeUnKnown notifyType = iota
	notifyTypeOutputTask
	notifyTypeDeleteTask
)

const (
	discardReasonNone          = iota
	discardReasonReachMaxRetry = -10000
)

// FileCollectRule defines the rules for collecting files in the archive process.
// It contains configuration options for how source files should be handled after archiving.
type FileCollectRule struct {
	KeepSourceFile    bool  `yaml:"keepSourceFile,omitempty" json:"keepSourceFile,omitempty"`
	ModifyProtectTime int64 `yaml:"modifyProtectTime,omitempty" json:"modifyProtectTime,omitempty"`
}

// Archive represents the main structure for file archiving operations.
// It contains configuration and runtime state for monitoring, uploading and managing files.
type Archive struct {
	PoolSize     int             `yaml:"poolSize,omitempty" json:"poolSize,omitempty"`
	Paths        []string        `yaml:"paths,omitempty" json:"paths,omitempty"`
	ExcludeFiles []string        `yaml:"excludeFiles,omitempty" json:"excludeFiles,omitempty"`
	CollectRule  FileCollectRule `yaml:"collectRule,omitempty" json:"collectRule,omitempty"`
	OutputRaw    json.RawMessage `yaml:"output,omitempty" json:"output,omitempty" filearchive:"namespace=output inline_key=type"`

	ctx       logarchive.Context
	fileCache fileCacheMap

	output logarchive.Outputter

	ticker  *time.Ticker
	watcher *fsnotify.Watcher
	logger  *zap.SugaredLogger
	regs    []*regexp.Regexp

	done       chan struct{}
	deleteChan chan *fileCacheKey
	notifyChan chan *notifyInfo
	tasks      chan func() error
}

type fileInfo struct {
	uploadFailedCount int
	deleteFailedCount int
	protectedEndTime  int64
	status            fileStatus
}

type notifyInfo struct {
	typ       notifyType
	watchPath string
	filePath  string
	result    bool
}

// ArchiveModule returns the file module information.
func (Archive) ArchiveModule() logarchive.ModuleInfo {
	return logarchive.ModuleInfo{
		ID: "file",
		New: func() logarchive.Module {
			return new(Archive)
		},
	}
}

// Provision implement the module interface
func (ar *Archive) Provision(ctx logarchive.Context) error {
	ar.ctx = ctx
	ar.logger = ctx.Logger().Sugar().Named("file")
	ar.ticker = time.NewTicker(time.Second)
	ar.fileCache = make(fileCacheMap)

	if ar.PoolSize == 0 {
		ar.PoolSize = 1
	}

	var err error

	// load output module
	mod, err := ctx.LoadModule(ar, "OutputRaw")
	if err != nil {
		return err
	}

	ar.output = mod.(logarchive.Outputter)

	if ar.watcher == nil {
		ar.watcher, err = fsnotify.NewWatcher()
		if err != nil {
			return fmt.Errorf("new watcher %v", err)
		}
	}

	if len(ar.ExcludeFiles) != 0 {
		for _, ex := range ar.ExcludeFiles {
			if re, err := regexp.Compile(ex); err != nil {
				return fmt.Errorf("invalid excelude file format: %v", err)
			} else {
				ar.regs = append(ar.regs, re)
			}
		}
	}

	ar.done = make(chan struct{})
	ar.tasks = make(chan func() error, 1000)
	ar.notifyChan = make(chan *notifyInfo, 100)
	ar.deleteChan = make(chan *fileCacheKey, 100)

	for _, rootPath := range ar.Paths {
		if walkErr := filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if !d.IsDir() {
				return nil
			}

			return ar.addWatchPath(rootPath, path)
		}); walkErr != nil {
			return walkErr
		}
	}
	return nil
}

// Validate implement the module interface
func (ar *Archive) Validate() error {
	for _, path := range ar.Paths {
		_, err := os.Stat(path)
		if err != nil {
			return err
		}
	}
	return nil
}

// Cleanup implement the module interface
func (ar *Archive) Cleanup() error {
	return nil
}

// Start implement the archive interface
func (ar *Archive) Start() error {
	// start output task
	for i := 0; i < ar.PoolSize; i++ {
		go ar.runOutputTask()
		if !ar.CollectRule.KeepSourceFile {
			go ar.runDeleteFileTask()
		}
	}

	go ar.run()
	return nil
}

// Stop implement the archive interface
func (ar *Archive) Stop() error {
	if ar.hasStopped() {
		return nil
	}

	close(ar.done)

	err := ar.watcher.Close()
	if err != nil {
		return err
	}
	return nil
}

func (ar *Archive) hasStopped() bool {
	select {
	case <-ar.done:
		return true
	default:
		return false
	}
}

func (ar *Archive) run() {
	for {
		select {
		case <-ar.ctx.Done():
			return
		case e, ok := <-ar.notifyChan:
			if e == nil || !ok {
				return
			}
			ar.handleTaskNotify(e)
		case event, ok := <-ar.watcher.Events:
			if !ok {
				return
			}

			ar.logger.Debugf("fs event notify name: %s event: %s", event.Name, event.Op.String())

			if err := ar.handleWatcherEvent(event); err != nil {
				ar.logger.Errorf("handle watcher event: %v", err)
			}
		case err, ok := <-ar.watcher.Errors:
			if !ok {
				return
			}
			ar.logger.Errorf("watcher error: %v", err)
		case t, ok := <-ar.ticker.C:
			if !ok {
				return
			}

			for _, p := range ar.Paths {
				usage, err := disk.Usage(p)
				if err != nil {
					continue
				}
				logarchive.DiskUsage.WithLabelValues(ar.ArchiveModule().ID.Name(), usage.Path, usage.Fstype).Set(usage.UsedPercent)
			}

			for watchPath, cache := range ar.fileCache {
				for k, v := range cache.files {
					if v.status != fileStatusWaitUpload || v.protectedEndTime > t.Unix() {
						continue
					}

					info, err := os.Stat(k)
					if err != nil {
						delete(cache.files, k)
						continue
					}

					protectedEndTime := info.ModTime().Unix() + ar.CollectRule.ModifyProtectTime
					if protectedEndTime > t.Unix() {
						v.protectedEndTime = protectedEndTime
						continue
					}

					if v.uploadFailedCount == 0 {
						logarchive.InputRequestSize.WithLabelValues(ar.ArchiveModule().ID.Name()).Observe(float64(info.Size()))
					}

					v.status = fileStatusUploading
					if !ar.trySubmitTask(func() error {
						task := ar.output.TaskInfo().New()
						err = ar.fillTaskInfo(task, cache.rootPath, k)
						if err != nil {
							ar.logger.Errorf("fill task info: %v", err)
							ar.notifyTaskExecuteResult(watchPath, k, false)
							return err
						}

						err = ar.output.Execute(task)
						if err != nil {
							ar.notifyTaskExecuteResult(watchPath, k, false)
							ar.logger.Errorf("execute input task failed: %v, filepath: %s", err, k)
							return err
						}

						ar.notifyTaskExecuteResult(watchPath, k, true)
						return err
					}) {
						v.status = fileStatusWaitUpload
					}
				}
			}

			logarchive.InputQueneSize.WithLabelValues(ar.ArchiveModule().ID.Name()).Set(float64(len(ar.tasks)))
		}
	}
}

func (ar *Archive) runOutputTask() {
	ar.logger.Debug("output task start")

	for {
		select {
		case <-ar.ctx.Done():
			return
		case <-ar.done:
			return
		case task, ok := <-ar.tasks:
			if task == nil || !ok {
				return
			}
			task()
		}
	}
}

func (ar *Archive) runDeleteFileTask() {
	ar.logger.Debug("delete file task start")

	for {
		select {
		case <-ar.ctx.Done():
			return
		case <-ar.done:
			return
		case e, ok := <-ar.deleteChan:
			if e == nil || !ok {
				return
			}

			defer releaseCacheKey(e)

			var result bool = false
			if err := os.Remove(e.filePath); err != nil {
				ar.logger.Errorf("remove file: %s got error: %v", e.filePath, err)
			} else {
				result = true
				ar.logger.Infof("file: %s has been removed successfully", e.filePath)
			}

			notify := newNotifyInfo(notifyTypeDeleteTask, e.watchPath, e.filePath, result)
			ar.sendNotify(notify)
		}
	}
}

func (ar *Archive) handleWatcherEvent(event fsnotify.Event) error {
	if event.Has(fsnotify.Remove) && !event.Has(fsnotify.Rename) {
		ar.removeCache(event.Name)
		return nil
	}

	// only care about the create event
	if !event.Has(fsnotify.Create) {
		return nil
	}

	info, err := os.Stat(event.Name)
	if err != nil {
		return err
	}

	// add new watch path
	if info.IsDir() {
		for _, r := range ar.Paths {
			if _, err := filepath.Rel(r, event.Name); err != nil {
				continue
			}
			return ar.addWatchPath(r, event.Name)
		}
		return fmt.Errorf("path: %s has no matched base path", event.Name)
	}

	// filter exculude files
	for _, re := range ar.regs {
		// skip execlude files
		if re.MatchString(event.Name) {
			return nil
		}
	}

	cache, ok := ar.fileCache[filepath.Dir(event.Name)]
	if !ok {
		return fmt.Errorf("watch path:%s not found", filepath.Dir(event.Name))
	}

	fi := &fileInfo{
		protectedEndTime: info.ModTime().Unix() + ar.CollectRule.ModifyProtectTime,
		status:           fileStatusWaitUpload,
	}
	cache.files[event.Name] = fi
	ar.logger.Debugf("file:%s has been add into watch list", event.Name)
	return nil
}

func (ar *Archive) handleTaskNotify(e *notifyInfo) {
	ar.logger.Debugf("task notify type: %d, watchpath:%s, filepath: %s, result: %v", e.typ, e.watchPath, e.filePath, e.result)
	defer releaseNotifyInfo(e)

	switch e.typ {
	case notifyTypeOutputTask:
		v, ok := ar.fileCache.getFile(e.watchPath, e.filePath)
		if !ok {
			break
		}

		if !e.result {
			v.uploadFailedCount++
			// last task execute failed, retry it
			if v.uploadFailedCount < 3 {
				v.status = fileStatusWaitUpload
				v.protectedEndTime = time.Now().Unix() + ar.CollectRule.ModifyProtectTime
				break
			}
		}

		if e.result {
			v.status = fileStatusUploaded
		} else {
			logarchive.InputDiscardTotal.WithLabelValues(ar.ArchiveModule().ID.Name(), strconv.Itoa(discardReasonReachMaxRetry)).Inc()
			ar.logger.Errorf("path: %v output task execute has failed %d times", e.filePath, v.uploadFailedCount)
		}

		if !ar.CollectRule.KeepSourceFile {
			key := newCacheKey(e.watchPath, e.filePath)
			ar.deleteChan <- key
		} else {
			ar.fileCache.removeFile(e.watchPath, e.filePath)
			ar.logger.Debugf("file:%s has been remove from watch list", e.filePath)
		}
	case notifyTypeDeleteTask:
		v, ok := ar.fileCache.getFile(e.watchPath, e.filePath)
		if !ok {
			break
		}

		if !e.result {
			v.deleteFailedCount++
			// try delete file again
			if v.deleteFailedCount < 3 {
				key := newCacheKey(e.watchPath, e.filePath)
				ar.deleteChan <- key
				break
			}
		}
		ar.fileCache.removeFile(e.watchPath, e.filePath)
		ar.logger.Debugf("file:%s has been remove from watch list", e.filePath)
	}
}

func (ar *Archive) notifyTaskExecuteResult(watchPath, filePath string, result bool) {
	notify := newNotifyInfo(notifyTypeOutputTask, watchPath, filePath, result)
	ar.sendNotify(notify)
}

func (ar *Archive) trySubmitTask(task func() error) (submitted bool) {
	select {
	case ar.tasks <- task:
		submitted = true
		return
	default:
		return
	}
}

func (ar *Archive) sendNotify(notify *notifyInfo) {
	if notify != nil {
		ar.notifyChan <- notify
	}
}

func (ar *Archive) removeCache(name string) {
	delete(ar.fileCache, name)
	//ar.logger.Warnf("path: %s has been removed from watch list", name)
}

func (ar *Archive) addWatchPath(root, name string) error {
	if _, ok := ar.fileCache[name]; ok {
		return nil
	}

	if watchErr := ar.watcher.AddWith(name); watchErr != nil {
		return watchErr
	}

	// TODO ignore unix.IN_MODIFY|unix.IN_ATTRIB

	cache := &element{
		rootPath: root,
		files:    make(map[string]*fileInfo),
	}

	// add historical files index
	if !ar.CollectRule.KeepSourceFile {
		if walkErr := filepath.WalkDir(name, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if d.IsDir() {
				if path == name {
					return nil
				}
				return filepath.SkipDir
			}

			// filter exculude files
			for _, re := range ar.regs {
				// skip execlude files
				if re.MatchString(path) {
					return nil
				}
			}

			if _, ok := cache.files[path]; !ok {
				info, err2 := d.Info()
				if err2 != nil {
					return err2
				}

				fi := &fileInfo{
					protectedEndTime: info.ModTime().Unix() + ar.CollectRule.ModifyProtectTime,
					status:           fileStatusWaitUpload,
				}
				cache.files[path] = fi
			}
			return nil
		}); walkErr != nil {
			return walkErr
		}
	}

	ar.fileCache[name] = cache
	ar.logger.Infof("path: %s has been add into watch list, root path: %s", name, root)
	return nil
}

func (ar *Archive) fillTaskInfo(task logarchive.OutputTask, rootPath, filePath string) error {
	switch t := task.(type) {
	case *cos.Task:
		t.RootPath = rootPath
		t.FilePath = filePath
		return nil
	default:
		return fmt.Errorf("unsupport output task type")
	}
}

func newNotifyInfo(typ notifyType, watchPath, filePath string, result bool) *notifyInfo {
	info := notifyPool.Get().(*notifyInfo)

	info.watchPath = watchPath
	info.filePath = filePath
	info.typ = typ
	info.result = result
	return info
}

func releaseNotifyInfo(info *notifyInfo) {
	if info == nil {
		return
	}

	info.watchPath = ""
	info.filePath = ""
	info.typ = notifyTypeUnKnown
	info.result = false
	notifyPool.Put(info)
}

func init() {
	logarchive.RegisterModule(Archive{})
}

var (
	notifyPool = sync.Pool{
		New: func() any {
			return new(notifyInfo)
		},
	}
)

var (
	_ logarchive.Provisioner  = (*Archive)(nil)
	_ logarchive.Validator    = (*Archive)(nil)
	_ logarchive.CleanerUpper = (*Archive)(nil)
)
