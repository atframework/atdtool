package cos

import "github.com/atframework/atdtool/internal/pkg/logarchive"

// Task represents a COS output task configuration
type Task struct {
	RootPath string `yaml:"rootPath,omitempty" json:"rootPath,omitempty"`
	FilePath string `yaml:"filePath,omitempty" json:"filePath,omitempty"`
}

// TaskInfo returns the OutputTaskInfo for COS task
// This method implements the logarchive.OutputTask interface
func (Task) TaskInfo() logarchive.OutputTaskInfo {
	return logarchive.OutputTaskInfo{
		New: func() logarchive.OutputTask {
			return new(Task)
		},
	}
}

var (
	_ logarchive.OutputTask = (*Task)(nil)
)
