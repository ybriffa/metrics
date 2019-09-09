package http

import (
	"errors"
	"sync"

	"github.com/rcrowley/go-metrics"
)

type section struct {
	name     string
	registry metrics.Registry

	m sync.RWMutex
}

func (s *section) setRegistry(registry metrics.Registry) {
	s.m.Lock()
	defer s.m.Unlock()

	s.registry = registry
}

func (s *section) getMetrics() (interface{}, error) {
	s.m.RLock()
	registry := s.registry
	s.m.RUnlock()

	if registry == nil {
		return nil, errors.New("nil registry")
	}

	return registry.GetAll(), nil
}
