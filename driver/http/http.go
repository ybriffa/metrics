package http

import (
	"encoding/json"
	"net/http"
	"sync"

	treemux "github.com/dimfeld/httptreemux"
	"github.com/ybriffa/metrics/driver"
)

type httpDriver struct {
	treemux  *treemux.TreeMux
	sections sync.Map
	m        sync.RWMutex
}

var (
	handler = &httpDriver{
		treemux: treemux.New(),
	}
)

func init() {
	// registers the metric
	driver.Register("http", driver.FactoryFunc(factory))

	handler.treemux.RedirectBehavior = treemux.UseHandler
	handler.treemux.Handle("GET", "/sections", handler.listSections)
	handler.treemux.Handle("GET", "/sections/metrics", handler.expandSections)
	handler.treemux.Handle("GET", "/section/:name", handler.showSection)
}

// factory is the function creating a new OpenTSDB Sender through the driver.Factory
func factory(name string) (driver.Driver, error) {
	return handler, nil
}

func (hd *httpDriver) listSections(w http.ResponseWriter, r *http.Request, _ map[string]string) {
	names := []string{}
	hd.sections.Range(func(k, _ interface{}) bool {
		names = append(names, k.(string))
		return true
	})

	e := json.NewEncoder(w)
	e.Encode(names)
}

func (hd *httpDriver) expandSections(w http.ResponseWriter, r *http.Request, _ map[string]string) {
	result := map[string]interface{}{}

	hd.sections.Range(func(k, v interface{}) bool {
		section := v.(*section)

		metrics, err := section.getMetrics()
		if err == nil {
			result[k.(string)] = metrics
		}
		return true
	})

	e := json.NewEncoder(w)
	e.Encode(result)
}

func (hd *httpDriver) showSection(w http.ResponseWriter, r *http.Request, args map[string]string) {
	sectionRaw, exists := hd.sections.Load(args["name"])
	if !exists {
		http.Error(w, "section not found", http.StatusNotFound)
		return
	}

	section, ok := sectionRaw.(*section)
	if !ok {
		http.Error(w, "object is not a section", http.StatusNotFound)
		return
	}

	m, err := section.getMetrics()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	e := json.NewEncoder(w)
	e.Encode(m)
}

func (hd *httpDriver) Send(registries []*driver.Registry) error {
	for _, registry := range registries {
		sectionRaw, _ := hd.sections.LoadOrStore(registry.Name, &section{
			name: registry.Name,
		})
		sectionRaw.(*section).setRegistry(registry.Registry)
	}
	return nil
}

func GetHandler() http.Handler {
	return handler.treemux
}
