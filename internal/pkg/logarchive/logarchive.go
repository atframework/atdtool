package logarchive

import (
	"context"
	"encoding/json"
	"fmt"
)

// Config is the top of the logarchive configuration structure.
type Config struct {
	Logging *Logging `yaml:"log,omitempty" json:"log,omitempty"`

	Metric *Metric `yaml:"metric,omitempty" json:"metric,omitempty"`

	ArchivesRaw ModuleMap `yaml:"archives,omitempty" json:"archives,omitempty"`

	archives map[string]Archive

	cancelFunc context.CancelFunc
}

// Start start the logarchive.
func Start(cfg []byte) error {
	var err error

	newCfg := new(Config)
	if err := json.Unmarshal(cfg, newCfg); err != nil {
		return err
	}

	ctx, err := run(newCfg)
	if err != nil {
		return err
	}

	logarchiveCtx = ctx
	return nil
}

func run(newCfg *Config) (Context, error) {
	var err error

	ctx, cancel := NewContext(Context{Context: context.Background(), cfg: newCfg})
	defer func() {
		if err != nil {
			// if there were any errors during startup,
			// we should cancel the new context we created
			cancel()
		}
	}()
	newCfg.cancelFunc = cancel

	if newCfg.Logging != nil {
		if err := newCfg.Logging.Provision(ctx); err != nil {
			return ctx, err
		}
	}

	if newCfg.Metric != nil {
		if err := newCfg.Metric.Provision(ctx); err != nil {
			return ctx, err
		}
	}

	newCfg.archives = make(map[string]Archive)

	// load archives
	err = func() error {
		for archiveName := range newCfg.ArchivesRaw {
			if _, err := ctx.Archive(archiveName); err != nil {
				return err
			}
		}
		return nil
	}()
	if err != nil {
		return ctx, err
	}

	// start archives
	err = func() error {
		started := make([]string, 0, len(newCfg.archives))
		for name, ar := range newCfg.archives {
			if err := ar.Start(); err != nil {
				for _, startedArchiveName := range started {
					if err2 := newCfg.archives[startedArchiveName].Stop(); err2 != nil {
						err = fmt.Errorf("%v; stop archive: %v",
							err, err2)
					}
				}
				return fmt.Errorf("archive start: %v", err)
			}
			started = append(started, name)
		}
		return nil
	}()

	// start record metric
	if newCfg.Metric != nil {
		err = newCfg.Metric.Start()
	}
	return ctx, err
}

// Stop stop the logarchive.
func Stop() error {
	logarchiveCtx.Logger().Sugar().Error("logarchive shutdown")

	if err := shutdown(logarchiveCtx); err != nil {
		fmt.Printf("logarchive shutdown: %v", err)
		return err
	}

	logarchiveCtx = Context{}
	return nil
}

func shutdown(ctx Context) error {
	if ctx.cfg == nil {
		return nil
	}

	var err error
	// stop metric
	if ctx.cfg.Metric != nil {
		if err2 := ctx.cfg.Metric.Stop(); err2 != nil {
			err = fmt.Errorf("%v; stop metric: %v", err, err2)
		}
	}

	// stop archives
	for _, s := range ctx.cfg.archives {
		if err2 := s.Stop(); err2 != nil {
			err = fmt.Errorf("%v; stop archive: %v", err, err2)
		}
	}

	ctx.cfg.cancelFunc()
	return err
}

// Archive is an interface that defines the basic operations for file archives.
// Implementations should provide Start and Stop methods to manage the archive lifecycle.
type Archive interface {
	Start() error
	Stop() error
}

// OutputTask is an interface that defines the basic operations for output tasks.
// Implementations should provide TaskInfo method to get task information.
type OutputTask interface {
	TaskInfo() OutputTaskInfo
}

// OutputTaskInfo defines the structure containing information about an output task.
// It provides a factory function to create new instances of OutputTask.
type OutputTaskInfo struct {
	New func() OutputTask
}

// Outputter is an interface that defines the contract for output operations.
// Implementations must provide methods to get task information and execute output tasks.
type Outputter interface {
	TaskInfo() OutputTaskInfo
	Execute(OutputTask) error
}

var (
	// logarchiveCtx is root context
	logarchiveCtx Context
)
