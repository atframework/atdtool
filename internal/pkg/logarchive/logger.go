package logarchive

import (
	"io"
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// CustomLog represents a custom logger configuration.
type CustomLog struct {
	Level    string `yaml:"level,omitempty" json:"level,omitempty"`
	Path     string `yaml:"path,omitempty" json:"path,omitempty"`
	RollSize int    `yaml:"rollSize,omitempty" json:"rollSize,omitempty"`
	RollKeep int    `yaml:"rollKeep,omitempty" json:"rollKeep,omitempty"`

	writer io.WriteCloser

	core         zapcore.Core
	encoder      zapcore.Encoder
	dynamicLevel zap.AtomicLevel
}

func (l *CustomLog) provision(_ Context) error {
	var err error
	l.dynamicLevel, err = zap.ParseAtomicLevel(l.Level)
	if err != nil {
		return err
	}

	if l.Path != "" {
		writer := &lumberjack.Logger{
			Filename:   l.Path,
			MaxSize:    l.RollSize,
			MaxBackups: l.RollKeep,
			LocalTime:  true,
			Compress:   false,
		}
		l.writer = writer
	} else {
		l.writer = os.Stdout
	}

	if l.encoder == nil {
		l.encoder = newDefaultLogEncoder()
	}

	l.buildCore()
	return nil
}

func (l *CustomLog) buildCore() {
	core := zapcore.NewCore(l.encoder, zapcore.AddSync(l.writer), l.dynamicLevel)
	l.core = core
}

// Logging is default logger for firearchive.
type Logging struct {
	CustomLog `yaml:",inline" json:",inline"`
	logger    *zap.Logger
}

// Provision initializes the Logging instance by provisioning the underlying CustomLog
// and creating a new zap.Logger instance
func (l *Logging) Provision(ctx Context) error {
	if err := l.CustomLog.provision(ctx); err != nil {
		return err
	}

	logger := zap.New(l.CustomLog.core)

	// capture logs from other libraries which
	// may not be using zap logging directly
	_ = zap.RedirectStdLog(logger)

	l.logger = logger
	return nil
}

func newDefaultLogEncoder() zapcore.Encoder {
	encCfg := zap.NewProductionEncoderConfig()
	// if interactive terminal, make output more human-readable by default
	encCfg.EncodeTime = func(ts time.Time, encoder zapcore.PrimitiveArrayEncoder) {
		encoder.AppendString(ts.Local().Format("2006/01/02 15:04:05.000"))
	}
	return zapcore.NewConsoleEncoder(encCfg)
}
