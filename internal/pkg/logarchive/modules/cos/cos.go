package cos

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/atframework/atdtool/internal/pkg/logarchive"
	"github.com/atframework/atdtool/pkg/compress"
	"github.com/tencentyun/cos-go-sdk-v5"
	"go.uber.org/zap"
)

// Status codes for COS operations
const (
	codeSuccess        int = iota
	codeInvalidParam       = -10000
	codeCallAPIFailed      = -10001
	codeCompressFailed     = -10002
)

type ArchiveRule string

const (
	EmptyArchive  ArchiveRule = ""
	HourArchive   ArchiveRule = "hour"
	DayArchive    ArchiveRule = "day"
	MonthArchive  ArchiveRule = "month"
	YearArchive   ArchiveRule = "year"
	CustomArchive ArchiveRule = "custom"
)

// FileUploadRule defines rules for file uploads to COS
type FileUploadRule struct {
	ArchiveRule       ArchiveRule                `yaml:"archiveRule,omitempty" json:"archiveRule,omitempty"`
	CompressAlgorithm compress.CompressAlgorithm `yaml:"compress,omitempty" json:"compress,omitempty"`
	MaxFileSize       int                        `yaml:"maxFileSize,omitempty" json:"maxFileSize,omitempty"`
	Timeout           int64                      `yaml:"timeout,omitempty" json:"timeout,omitempty"`
}

// Handler implements COS file archiving functionality
type Handler struct {
	Url        string         `yaml:"url,omitempty" json:"url,omitempty"`
	SecretID   string         `yaml:"secretID,omitempty" json:"secretID,omitempty"`
	SecretKey  string         `yaml:"secretKey,omitempty" json:"secretKey,omitempty"`
	UploadRule FileUploadRule `yaml:"uploadRule,omitempty" json:"uploadRule,omitempty"`

	ctx logarchive.Context

	task   logarchive.OutputTaskInfo
	client *cos.Client

	logger *zap.SugaredLogger
}

// ArchiveModule returns the cos output module information.
func (Handler) ArchiveModule() logarchive.ModuleInfo {
	return logarchive.ModuleInfo{
		ID: "output.cos",
		New: func() logarchive.Module {
			return new(Handler)
		},
	}
}

// Provision implement the output interface
func (h *Handler) Provision(ctx logarchive.Context) error {
	h.ctx = ctx
	h.logger = ctx.Logger().Sugar().Named("cos")
	h.task = (Task{}).TaskInfo()

	url, _ := url.Parse(h.Url)
	bktUrl := &cos.BaseURL{BucketURL: url}

	if h.client == nil {
		h.client = cos.NewClient(bktUrl, &http.Client{
			Transport: &cos.AuthorizationTransport{
				SecretID:  h.SecretID,
				SecretKey: h.SecretKey,
			},
		})
	}
	return nil
}

// Validate implement the output interface
func (h *Handler) Validate() error {
	if h.client == nil {
		return fmt.Errorf("invalid cos client")
	}

	ok, err := h.client.Bucket.IsExist(h.ctx)
	if err != nil {
		return fmt.Errorf("check cos bucket: %v", err)
	}

	if !ok {
		return fmt.Errorf("cos bucket does not exist")
	}
	return nil
}

// Cleanup implement the output interface
func (h *Handler) Cleanup() error {
	return nil
}

func (h *Handler) TaskInfo() logarchive.OutputTaskInfo {
	return h.task
}

// Handle implement the output interface
func (h *Handler) Execute(t logarchive.OutputTask) error {
	var errCode int = codeSuccess

	begin := time.Now()
	defer func() {
		logarchive.OutputRequestTotal.WithLabelValues(h.ArchiveModule().ID.Name(), strconv.Itoa(errCode)).Inc()
		logarchive.OutputRequestDuration.WithLabelValues(h.ArchiveModule().ID.Name(), strconv.Itoa(errCode)).Observe(float64(time.Since(begin).Seconds()))
	}()

	task, ok := t.(*Task)
	if !ok {
		errCode = codeInvalidParam
		return fmt.Errorf("invalid cos output task")
	}

	info, err := os.Stat(task.FilePath)
	if err != nil {
		errCode = codeInvalidParam
		h.logger.Errorf("cos upload stat file: %s failed: %v", task.FilePath, err)
		return err
	}

	if info.IsDir() {
		errCode = codeInvalidParam
		h.logger.Errorf("cos upload file: %s is directory", task.FilePath)
		return fmt.Errorf("input: %s is directory", task.FilePath)
	}

	dstPath, err := filepath.Rel(task.RootPath, task.FilePath)
	if err != nil {
		h.logger.Errorf("can't get targetpath: %s relative path to basepath: %s for reason: %v", task.FilePath, task.RootPath, err)
		return err
	}

	prefix := getArchivePrefix(h.UploadRule.ArchiveRule, task.FilePath)
	if prefix != "" {
		dstPath = filepath.Join(prefix, dstPath)
	}

	// add suffix by compress type
	dstPath += compress.GetCompressAlgorithmSuffix(h.UploadRule.CompressAlgorithm)

	// use cos advanced api
	if h.UploadRule.CompressAlgorithm == compress.NONE {
		_, _, err = h.client.Object.Upload(h.ctx, dstPath, task.FilePath, nil)
		if err != nil {
			errCode = codeCallAPIFailed
			h.logger.Errorf("call upload api: %v", err)
		}
		return err
	}

	// compress target file
	buf := newCompressBuffer()
	defer freeCompressBuffer(buf)

	err = compress.CompressFile(task.FilePath, compress.NewDefaultCompressOption(h.UploadRule.CompressAlgorithm), buf)
	if err != nil && err != compress.ErrUnexpectedEOF {
		errCode = codeCompressFailed
		h.logger.Errorf("compress file: %s failed: %v", task.FilePath, err)
		return err
	}

	if err == compress.ErrUnexpectedEOF {
		logarchive.OutputTruncateTotal.WithLabelValues(h.ArchiveModule().ID.Name()).Inc()
		h.logger.Warnf("file %s size %d is too larger", task.FilePath, info.Size())
	}

	_, err = h.client.Object.Put(h.ctx, dstPath, buf, nil)
	if err != nil {
		errCode = codeCallAPIFailed
		h.logger.Errorf("call upload api: %v", err)
		return err
	}
	return nil
}

func getArchivePrefix(rule ArchiveRule, in string) string {
	var modifyTime time.Time

	info, err := os.Stat(in)
	if err != nil {
		modifyTime = time.Now()
	} else {
		modifyTime = info.ModTime()
	}

	switch rule {
	case HourArchive:
		return modifyTime.Format("2006010215")
	case DayArchive:
		return modifyTime.Format("20060102")
	case MonthArchive:
		return modifyTime.Format("200601")
	case YearArchive:
		return modifyTime.Format("2006")
	default:
		return ""
	}
}

func newCompressBuffer() *bytes.Buffer {
	buf := compressBufferPool.Get().(*bytes.Buffer)
	return buf
}

func freeCompressBuffer(buf *bytes.Buffer) {
	if buf == nil || buf.Len() > 1024*1024 {
		return
	}
	buf.Reset()
	compressBufferPool.Put(buf)
}

func init() {
	logarchive.RegisterModule(Handler{})
}

var (
	// compressBufPool is used for buffering compressed data.
	compressBufferPool = sync.Pool{
		New: func() any {
			return new(bytes.Buffer)
		},
	}
)

var (
	_ logarchive.Provisioner  = (*Handler)(nil)
	_ logarchive.Validator    = (*Handler)(nil)
	_ logarchive.CleanerUpper = (*Handler)(nil)
	_ logarchive.Outputter    = (*Handler)(nil)
)
