package http

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/rcrowley/go-metrics"
)

var nameReplacer = strings.NewReplacer("-", "_", ".", "_")

type section struct {
	name     string
	registry metrics.Registry
	tags     map[string]string

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

	return s.registry.GetAll(), nil
}

func (s *section) getGTS() ([]*GTS, error) {
	s.m.RLock()
	registry := s.registry
	s.m.RUnlock()

	if registry == nil {
		return nil, errors.New("nil registry")
	}

	series := []*GTS{}
	now := time.Now().UnixNano() / int64(time.Millisecond)
	var errs []error
	registry.Each(func(name string, i interface{}) {
		name = fmt.Sprintf("%s_%s", s.name, name)
		name = nameReplacer.Replace(strings.ToLower(name))
		newSeries, err := gtsFromMetric(name, i, now, s.tags)
		if err != nil {
			errs = append(errs, err)
			return
		}
		series = append(series, newSeries...)
	})

	if len(errs) > 0 {
		return nil, errs[0]
	}

	return series, nil
}
