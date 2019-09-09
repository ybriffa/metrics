package http

import (
	"encoding/json"
	"net/http"
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

func (s *section) serveHTTP(w http.ResponseWriter, r *http.Request) {
	s.m.RLock()
	registry := s.registry
	s.m.RUnlock()

	if registry == nil {
		http.Error(w, "nil registry", http.StatusInternalServerError)
		return
	}

	content, err := json.Marshal(registry.GetAll())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(content)
}
