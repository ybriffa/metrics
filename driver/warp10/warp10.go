package warp10

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/eapache/go-resiliency/retrier"
	"github.com/ovh/configstore"
	"github.com/rcrowley/go-metrics"
	log "github.com/sirupsen/logrus"
	"github.com/ybriffa/metrics/driver"
)

const (
	configStoreAlias = "warp10-metrics"
)

func init() {
	// registers the metric
	driver.Register("warp10", driver.FactoryFunc(factory))
}

// factory is the function creating a new OpenTSDB Sender through the driver.Factory
func factory(name string) (driver.Driver, error) {
	rawConfig, err := configstore.GetItemValue(configStoreAlias)
	if err != nil {
		log.Info(err)
		if _, ok := err.(configstore.ErrItemNotFound); !ok {
			return nil, err
		}
		return nil, driver.ErrDriverDisabled
	}

	var w Warp10Sender
	if err = json.Unmarshal([]byte(rawConfig), &w); err != nil {
		return nil, err
	}
	w.applicationName = name
	return &w, w.Valid()
}

// Warp10Sender is the implementation of driver.Driver
type Warp10Sender struct {
	Address         string `json:"address"`
	Token           string `json:"token"`
	Prefix          string `json:"prefix"`
	applicationName string
}

// Valid defines whether or not the warp10 sender is valid
func (ws *Warp10Sender) Valid() error {
	if ws.Address == "" {
		return errors.New("address is empty")
	}

	if ws.Token == "" {
		return errors.New("token is empty")
	}

	if ws.Prefix == "" {
		ws.Prefix = ws.applicationName
	} else if ws.applicationName != "" {
		ws.Prefix = fmt.Sprintf("%s.%s", ws.Prefix, ws.applicationName)
	}
	return nil
}

// Send sends the registry metrics to opentsdb
func (ws *Warp10Sender) Send(registries []*driver.Registry) error {
	now := time.Now().UTC().UnixNano() / int64(time.Microsecond)

	series := []*GTS{}
	for _, registry := range registries {
		registry.Registry.Each(func(name string, i interface{}) {
			series = append(series, ws.writeMetric(fmt.Sprintf("%s.%s", registry.Name, name), i, float64(now), registry.Tags)...)
		})
	}

	if len(series) > 0 {
		req, err := ws.craftRequest(series)
		if err != nil {
			return err
		}

		r := retrier.New(retrier.ConstantBackoff(3, time.Second), nil)

		err = r.Run(func() error {
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				log.Errorf("[metrics] can't reach the warp10 backend: %s", err)
				return err
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				log.Warningf("[metrics] warp10 backend responded with %s", resp.Status)
			}
			return nil
		})

		if err != nil {
			return err
		}
	}

	return nil
}

func (ws *Warp10Sender) craftRequest(series []*GTS) (*http.Request, error) {
	var gtsArray [][]byte
	for _, serie := range series {
		gtsArray = append(gtsArray, serie.Encode())
	}
	body := bytes.Join(gtsArray, []byte(""))

	req, err := http.NewRequest("POST", ws.Address, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("X-Warp10-Token", ws.Token)

	return req, nil
}

// writeMetrics returns an array of metrics related to the type of metric given
func (ws *Warp10Sender) writeMetric(name string, i interface{}, now float64, tags map[string]string) []*GTS {
	m := []*GTS{}
	du := float64(time.Nanosecond)

	switch metric := i.(type) {

	case metrics.Counter:
		m = append(m, &GTS{Name: fmt.Sprintf("%s.%s.count", ws.Prefix, name), Ts: now, Value: metric.Count(), Labels: tags})

	case metrics.Gauge:
		m = append(m, &GTS{Name: fmt.Sprintf("%s.%s.value", ws.Prefix, name), Ts: now, Value: metric.Value(), Labels: tags})

	case metrics.GaugeFloat64:
		m = append(m, &GTS{Name: fmt.Sprintf("%s.%s.value", ws.Prefix, name), Ts: now, Value: metric.Value(), Labels: tags})

	case metrics.Histogram:
		h := metric.Snapshot()
		ps := h.Percentiles([]float64{0.5, 0.75, 0.95, 0.99, 0.999})
		m = append(m, &GTS{Name: fmt.Sprintf("%s.%s.count", ws.Prefix, name), Ts: now, Value: h.Count(), Labels: tags})
		m = append(m, &GTS{Name: fmt.Sprintf("%s.%s.min", ws.Prefix, name), Ts: now, Value: h.Min(), Labels: tags})
		m = append(m, &GTS{Name: fmt.Sprintf("%s.%s.max", ws.Prefix, name), Ts: now, Value: h.Max(), Labels: tags})
		m = append(m, &GTS{Name: fmt.Sprintf("%s.%s.mean", ws.Prefix, name), Ts: now, Value: h.Mean(), Labels: tags})
		m = append(m, &GTS{Name: fmt.Sprintf("%s.%s.std-dev", ws.Prefix, name), Ts: now, Value: h.StdDev(), Labels: tags})
		m = append(m, &GTS{Name: fmt.Sprintf("%s.%s.50-percentile", ws.Prefix, name), Ts: now, Value: ps[0], Labels: tags})
		m = append(m, &GTS{Name: fmt.Sprintf("%s.%s.75-percentile", ws.Prefix, name), Ts: now, Value: ps[1], Labels: tags})
		m = append(m, &GTS{Name: fmt.Sprintf("%s.%s.95-percentile", ws.Prefix, name), Ts: now, Value: ps[2], Labels: tags})
		m = append(m, &GTS{Name: fmt.Sprintf("%s.%s.99-percentile", ws.Prefix, name), Ts: now, Value: ps[3], Labels: tags})
		m = append(m, &GTS{Name: fmt.Sprintf("%s.%s.999-percentile", ws.Prefix, name), Ts: now, Value: ps[4], Labels: tags})

	case metrics.Meter:
		meter := metric.Snapshot()
		m = append(m, &GTS{Name: fmt.Sprintf("%s.%s.count", ws.Prefix, name), Ts: now, Value: meter.Count(), Labels: tags})
		m = append(m, &GTS{Name: fmt.Sprintf("%s.%s.one-minute", ws.Prefix, name), Ts: now, Value: meter.Rate1(), Labels: tags})
		m = append(m, &GTS{Name: fmt.Sprintf("%s.%s.five-minute", ws.Prefix, name), Ts: now, Value: meter.Rate5(), Labels: tags})
		m = append(m, &GTS{Name: fmt.Sprintf("%s.%s.fifteen-minute", ws.Prefix, name), Ts: now, Value: meter.Rate15(), Labels: tags})
		m = append(m, &GTS{Name: fmt.Sprintf("%s.%s.mean", ws.Prefix, name), Ts: now, Value: meter.RateMean(), Labels: tags})

	case metrics.Timer:
		t := metric.Snapshot()
		ps := t.Percentiles([]float64{0.5, 0.75, 0.95, 0.99, 0.999})
		m = append(m, &GTS{Name: fmt.Sprintf("%s.%s.count", ws.Prefix, name), Ts: now, Value: t.Count(), Labels: tags})
		m = append(m, &GTS{Name: fmt.Sprintf("%s.%s.min", ws.Prefix, name), Ts: now, Value: t.Min() / int64(du), Labels: tags})
		m = append(m, &GTS{Name: fmt.Sprintf("%s.%s.max", ws.Prefix, name), Ts: now, Value: t.Max() / int64(du), Labels: tags})
		m = append(m, &GTS{Name: fmt.Sprintf("%s.%s.mean", ws.Prefix, name), Ts: now, Value: t.Mean() / du, Labels: tags})
		m = append(m, &GTS{Name: fmt.Sprintf("%s.%s.std-dev", ws.Prefix, name), Ts: now, Value: t.StdDev() / du, Labels: tags})
		m = append(m, &GTS{Name: fmt.Sprintf("%s.%s.50-percentile", ws.Prefix, name), Ts: now, Value: ps[0] / du, Labels: tags})
		m = append(m, &GTS{Name: fmt.Sprintf("%s.%s.75-percentile", ws.Prefix, name), Ts: now, Value: ps[1] / du, Labels: tags})
		m = append(m, &GTS{Name: fmt.Sprintf("%s.%s.95-percentile", ws.Prefix, name), Ts: now, Value: ps[2] / du, Labels: tags})
		m = append(m, &GTS{Name: fmt.Sprintf("%s.%s.99-percentile", ws.Prefix, name), Ts: now, Value: ps[3] / du, Labels: tags})
		m = append(m, &GTS{Name: fmt.Sprintf("%s.%s.999-percentile", ws.Prefix, name), Ts: now, Value: ps[4] / du, Labels: tags})
		m = append(m, &GTS{Name: fmt.Sprintf("%s.%s.one-minute", ws.Prefix, name), Ts: now, Value: t.Rate1(), Labels: tags})
		m = append(m, &GTS{Name: fmt.Sprintf("%s.%s.five-minute", ws.Prefix, name), Ts: now, Value: t.Rate5(), Labels: tags})
		m = append(m, &GTS{Name: fmt.Sprintf("%s.%s.fifteen-minute", ws.Prefix, name), Ts: now, Value: t.Rate15(), Labels: tags})
		m = append(m, &GTS{Name: fmt.Sprintf("%s.%s.mean-rate", ws.Prefix, name), Ts: now, Value: t.RateMean(), Labels: tags})

	default:
		log.Errorf("Unknown metric type %T for metric '%s'", i, name)
	}

	return m
}
