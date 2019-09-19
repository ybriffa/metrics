package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
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

func (hd *httpDriver) expandSections(w http.ResponseWriter, r *http.Request, m map[string]string) {
	accept := r.Header.Get("Accept")
	if strings.ToLower(accept) == "application/prometheus" {
		hd.expandSectionsPrometheus(w, r, m)
		return
	}

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

func (hd *httpDriver) expandSectionsPrometheus(w http.ResponseWriter, r *http.Request, m map[string]string) {
	var itError error

	metrics := []*GTS{}
	hd.sections.Range(func(k, v interface{}) bool {
		section := v.(*section)

		gts, err := section.getGTS()
		if err != nil {
			itError = err
			return false
		}

		metrics = append(metrics, gts...)
		return true
	})

	if itError != nil {
		http.Error(w, itError.Error(), http.StatusInternalServerError)
		return
	}

	var b bytes.Buffer
	sort.SliceStable(metrics, func(i, j int) bool {
		return metrics[i].Name < metrics[j].Name
	})
	for _, metric := range metrics {
		b.Write(metric.Encode())
	}

	w.Write(b.Bytes())
}

func (hd *httpDriver) showSection(w http.ResponseWriter, r *http.Request, args map[string]string) {
	// Load the section from the map
	sectionRaw, exists := hd.sections.Load(args["name"])
	if !exists {
		http.Error(w, "section not found", http.StatusNotFound)
		return
	}

	// Cast it into its real type
	section, ok := sectionRaw.(*section)
	if !ok {
		http.Error(w, "object is not a section", http.StatusNotFound)
		return
	}

	// Get the metrics
	m, err := section.getMetrics()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Encode them as json
	e := json.NewEncoder(w)
	e.Encode(m)
}

// Send is the implementation of the driver.Registry.Sent. It exposes
// the registry given and deletes the old registries not declared in this array
func (hd *httpDriver) Send(registries []*driver.Registry) error {
	// First, range over all the registries to either create the entry or
	// update the metrics.Registry of the section.
	var registriesSent []string
	for _, registry := range registries {
		id := hd.computeSectionID(registry.Name, registry.Tags)
		sectionRaw, loaded := hd.sections.LoadOrStore(id, &section{
			name:     registry.Name,
			registry: registry.Registry,
			tags:     registry.Tags,
		})
		// If the section already existed, update its metrics registry
		if loaded {
			sectionRaw.(*section).setRegistry(registry.Registry)
		}
		// Save the name of the section to know which one to delete after
		registriesSent = append(registriesSent, id)
	}

	// Range over all the sections to compare the sections names
	// with the new one sent. All the sections not matching the sent
	// registries must be deleted
	var registriesToDelete []string
	hd.sections.Range(func(k, _ interface{}) bool {
		s := k.(string)
		for _, name := range registriesSent {
			if name == s {
				return true
			}
		}
		// save the names to delete if no match in the registries sent
		registriesToDelete = append(registriesToDelete, s)
		return true
	})

	// Delete the sections
	for _, name := range registriesToDelete {
		hd.sections.Delete(name)
	}

	return nil
}

func (hd *httpDriver) computeSectionID(name string, rawTags map[string]string) string {
	tags := []string{}
	for k, v := range rawTags {
		tags = append(tags, fmt.Sprintf("%s:%s", k, v))
	}
	sort.Stable(sort.StringSlice(tags))
	return fmt.Sprintf("%s(%s)", name, strings.Join(tags, ","))
}

// GetHandler returns the HTTP handler to register to expose the metrics
func GetHandler() http.Handler {
	return handler.treemux
}
