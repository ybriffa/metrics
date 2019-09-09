package driver

import (
	"errors"

	"github.com/rcrowley/go-metrics"
)

// ErrDriverDisabled is the error returned by the driver when trying to instanciate it
// while its configuration is not provided.
var ErrDriverDisabled error = errors.New("this driver is not enabled")

type Registry struct {
	Name     string
	Registry metrics.Registry
	Tags     map[string]string
}

// Driver manages the metrics to either push or expose them.
type Driver interface {
	Send([]*Registry) error
}
