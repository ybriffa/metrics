package logrus

import (
	"fmt"

	"github.com/rcrowley/go-metrics"
	log "github.com/sirupsen/logrus"
	"github.com/ybriffa/metrics/driver"
)

func init() {
	driver.Register("logrus", driver.FactoryFunc(factory))
}

func factory(name string) (driver.Driver, error) {
	return &LogrusSender{
		logger: log.WithField("application", name),
	}, nil
}

type LogrusSender struct {
	logger *log.Entry
}

func (ls *LogrusSender) Send(registries []*driver.Registry) error {
	for _, registry := range registries {
		slog := ls.logger.WithField("registry", registry.Name)

		for key, value := range registry.Tags {
			slog = slog.WithField(key, value)
		}

		registry.Registry.Each(func(name string, i interface{}) {
			writeMetric(slog, name, i)
		})
	}
	return nil
}

func writeMetric(slog *log.Entry, name string, i interface{}) {
	switch metric := i.(type) {

	case metrics.Counter:
		slog = slog.WithField(name, metric.Count())

	case metrics.Gauge:
		slog = slog.WithField(name, metric.Value())

	case metrics.GaugeFloat64:
		slog = slog.WithField(name, metric.Value())

	case metrics.Histogram:
		h := metric.Snapshot()
		slog = slog.WithFields(log.Fields{
			fmt.Sprintf("%s.count", name):   h.Count(),
			fmt.Sprintf("%s.min", name):     h.Min(),
			fmt.Sprintf("%s.max", name):     h.Max(),
			fmt.Sprintf("%s.mean", name):    h.Mean(),
			fmt.Sprintf("%s.std-dev", name): h.StdDev(),
		})

	case metrics.Meter:
		meter := metric.Snapshot()
		slog = slog.WithField(name, meter.Count())

	case metrics.Timer:
		t := metric.Snapshot()
		slog = slog.WithFields(log.Fields{
			fmt.Sprintf("%s.count", name):   t.Count(),
			fmt.Sprintf("%s.min", name):     t.Min(),
			fmt.Sprintf("%s.max", name):     t.Max(),
			fmt.Sprintf("%s.mean", name):    t.Mean(),
			fmt.Sprintf("%s.std-dev", name): t.StdDev(),
		})

	default:
		slog.Errorf("Unknown metric type %T for metric '%s'", i, name)
		return
	}

	slog.Info("new metric")
}
