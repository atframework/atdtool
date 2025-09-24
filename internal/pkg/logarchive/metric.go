package logarchive

import (
	"os"
	"path/filepath"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"go.uber.org/zap"
)

const (
	LogArciveSubSystem       = "logarchive"
	DiskUsageKey             = "disk_usage"
	InputQueneSizeKey        = "input_queue_size"
	InputRequestSizeKey      = "input_request_size_bytes"
	InputDiscardTotalKey     = "input_discard_total"
	OutputTruncateTotalKey   = "output_truncate_total"
	OutputRequestTotalKey    = "output_request_total"
	OutputRequestDurationKey = "output_request_duration_seconds"
)

var (
	DiskUsage = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: LogArciveSubSystem,
			Name:      DiskUsageKey,
			Help:      "The disk usage of the file path that watched",
		},
		[]string{
			"module",
			"path",
			"fstype",
		},
	)

	InputQueneSize = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: LogArciveSubSystem,
			Name:      InputQueneSizeKey,
			Help:      "The size of input task queue",
		},
		[]string{
			"module",
		},
	)

	InputRequestSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Subsystem: LogArciveSubSystem,
			Name:      InputRequestSizeKey,
			Help:      "Size of the input target in bytes",
			Buckets:   []float64{1e6, 1e7, 2e7, 3e7, 5e7, 1e8, 5e8, 1e9},
		},
		[]string{
			"module",
		},
	)

	InputDiscardTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Subsystem: LogArciveSubSystem,
			Name:      InputDiscardTotalKey,
			Help:      "The number of input taregt has been discard",
		},
		[]string{
			"module",
			"reason",
		},
	)

	OutputTruncateTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Subsystem: LogArciveSubSystem,
			Name:      OutputTruncateTotalKey,
			Help:      "The number of output has been truncated",
		},
		[]string{
			"module",
		},
	)

	OutputRequestTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Subsystem: LogArciveSubSystem,
			Name:      OutputRequestTotalKey,
			Help:      "Call logarchive output module requests",
		},
		[]string{
			"module",
			"code",
		},
	)

	OutputRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Subsystem: LogArciveSubSystem,
			Name:      OutputRequestDurationKey,
			Help:      "Histogram of the time (in seconds) each request took",
			Buckets:   append(prometheus.DefBuckets, 30, 60),
		},
		[]string{
			"module",
			"code",
		},
	)
)

// Metric struct defines the configuration and runtime state for logarchive metrics collection.
// It contains fields for output path, scrape interval, and manages the metrics collection process.
type Metric struct {
	OutPath       string `yaml:"outPath,omitempty" json:"outPath,omitempty"`
	ScrapInterval int    `yaml:"scrapInterval,omitempty" json:"scrapInterval,omitempty"`

	done   chan struct{}
	ticker time.Ticker

	register *prometheus.Registry

	logger *zap.SugaredLogger
}

// Provision initializes the Metric instance with required components
func (m *Metric) Provision(ctx Context) error {
	m.done = make(chan struct{})
	m.logger = ctx.Logger().Sugar().Named("metric")
	m.register = prometheus.NewRegistry()

	m.register.MustRegister(DiskUsage)
	m.register.MustRegister(InputQueneSize)
	m.register.MustRegister(InputRequestSize)
	m.register.MustRegister(InputDiscardTotal)
	m.register.MustRegister(OutputTruncateTotal)
	m.register.MustRegister(OutputRequestTotal)
	m.register.MustRegister(OutputRequestDuration)

	if m.ScrapInterval == 0 {
		m.ScrapInterval = 60
	}
	m.ticker = *time.NewTicker(time.Second * time.Duration(m.ScrapInterval))
	return nil
}

func (m *Metric) Start() error {
	go m.runRecordMetrics()
	return nil
}

func (m *Metric) Stop() error {
	if m.hasStopped() {
		return nil
	}

	close(m.done)
	return nil
}

func (m *Metric) hasStopped() bool {
	select {
	case <-m.done:
		return true
	default:
		return false
	}
}

// GetGather returns the gatherer. It used by test case outside current package.
func (m *Metric) GetGather() ([]*dto.MetricFamily, error) {
	return m.register.Gather()
}

func (m *Metric) runRecordMetrics() {
	fd, err := os.OpenFile(filepath.Join(m.OutPath, "logarchive.prom"), os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0644)
	if err != nil {
		panic(err)
	}

	defer fd.Close()

	for {
		select {
		case <-m.done:
			return
		case _, ok := <-m.ticker.C:
			if !ok {
				return
			}

			fd.Truncate(0)
			fd.Seek(0, 0)
			mfs, _ := m.GetGather()
			for _, mf := range mfs {
				expfmt.MetricFamilyToText(fd, mf)
			}

			m.logger.Info("metric info has been updated")
		}
	}
}
