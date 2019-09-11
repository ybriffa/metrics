package metrics

import (
	"fmt"
	"time"

	"github.com/rcrowley/go-metrics"
	log "github.com/sirupsen/logrus"
	"github.com/ybriffa/metrics/driver"
)

var (
	defaultManager *manager = &manager{
		registers:     make(map[string]*driver.Registry),
		flushInterval: time.Minute,
	}
)

// Init starts metrics sending
// It goes through all the drivers registered to the driver.Factory and tries to instanciate them
// If the sender is not enabled, it must returns ErrSenderDisabled.
// Every other error must be handled and the metrics sending will not start
func Init(appName string) error {
	for _, driverName := range driver.Registered() {
		s, err := driver.New(driverName, appName)
		if err != nil && err != driver.ErrDriverDisabled {
			return fmt.Errorf("failed to init metrics driver %s : %s", driverName, err)
		}
		if s != nil {
			log.Debugf("[metrics] sender %s init", driverName)
			defaultManager.senders = append(defaultManager.senders, s)
		}
	}
	go defaultManager.run()
	return nil
}

// Register adds a metrics.Registry to watch and send.
// It will send all the metrics in through all the senders init until it has been unregistered
func Register(name string, r metrics.Registry, tags map[string]string) {
	defaultManager.addRegistry(name, r, tags)
}

// RegisterStruct takes a pointer to a struct containing metrics, creates a metrics.Registry and Registers it
func RegisterStruct(name string, s interface{}, tags map[string]string) (metrics.Registry, error) {
	r, err := RegistryFromStruct(s)
	if err != nil {
		return nil, err
	}

	Register(name, r, tags)
	return r, nil
}

// Unregister deletes the metrics.Registry to the list of the registry watched
func Unregister(name string, tags map[string]string) error {
	return defaultManager.deleteRegistry(name, tags)
}

// FlushInterval sets the flush duration for the default manager
func FlushInterval(d time.Duration) {
	defaultManager.setFlushInterval(d)
}

// Flush triggers the send of the metrics to the drivers
func Flush() {
	defaultManager.flush()
}

// Stop stops all the senders inited
func Stop() {
	defaultManager.stop()
}
