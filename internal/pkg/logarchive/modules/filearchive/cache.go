package filearchive

import "sync"

type element struct {
	rootPath string
	files    map[string]*fileInfo
}

type fileCacheKey struct {
	watchPath string
	filePath  string
}

type fileCacheMap map[string]*element

func (m fileCacheMap) getFile(watchPath, filePath string) (*fileInfo, bool) {
	if c, ok := m[watchPath]; ok {
		if v, ok := c.files[filePath]; ok {
			return v, true
		}
	}
	return nil, false
}

func (m fileCacheMap) removeFile(watchPath, filePath string) {
	if c, ok := m[watchPath]; ok {
		delete(c.files, filePath)
	}
}

func newCacheKey(watchPath, filePath string) *fileCacheKey {
	key := cacheKeyPool.Get().(*fileCacheKey)

	key.watchPath = watchPath
	key.filePath = filePath
	return key
}

func releaseCacheKey(key *fileCacheKey) {
	if key == nil {
		return
	}

	key.watchPath = ""
	key.filePath = ""
	notifyPool.Put(key)
}

var (
	cacheKeyPool = sync.Pool{
		New: func() any {
			return new(fileCacheKey)
		},
	}
)
