package driver

import (
	"errors"
)

// Factory is the interface allowing to create a registered implementation of a Driver
type Factory interface {
	New(string) (Driver, error)
}

// FactoryFunc is an adapter to use functions to be registered as a Factory
type FactoryFunc func(string) (Driver, error)

// New is the implemention of the Factory.New
func (f FactoryFunc) New(name string) (Driver, error) {
	return f(name)
}

// factory is the map where Factory need to register
// each Factory should be register in its own init() function
var factory map[string]Factory = make(map[string]Factory)

// Register saves a Factory to be instanciated with the New function.
func Register(name string, f Factory) {
	factory[name] = f
}

// New instanciates a driver with the given application name. The driver must have been
// registered with the Register function.
func New(name, application string) (Driver, error) {
	f, exists := factory[name]
	if !exists {
		return nil, errors.New("unknown Factory type")
	}
	return f.New(application)
}

// Registered list all registered drivers.
func Registered() []string {
	ret := []string{}
	for key := range factory {
		ret = append(ret, key)
	}
	return ret
}
