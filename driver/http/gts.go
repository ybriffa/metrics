package http

import (
	"bytes"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/rcrowley/go-metrics"
)

// GTS struct
type GTS struct {
	Ts     int64
	Name   string
	Labels map[string]string
	Value  interface{}
}

// Encode a GTS to the Sensision format
// TS/LAT:LON/ELEV NAME{LABELS} VALUE
func (gts *GTS) Encode() []byte {
	var b bytes.Buffer

	b.WriteString(gts.Name)
	if len(gts.Labels) > 0 {
		b.WriteString("{")
		first := true
		for k, v := range gts.Labels {
			if !first {
				b.WriteString(",")
			}
			b.WriteString(k)
			b.WriteString(`="`)
			b.WriteString(v)
			b.WriteString(`"`)
			first = false
		}
		b.WriteString("}")
	}

	// value
	var value string
	switch gts.Value.(type) {
	case bool:
		if gts.Value.(bool) {
			value = "T"
		} else {
			value = "F"
		}

	case float64:
		value = fmt.Sprintf("%f", gts.Value.(float64))

	case float32:
		value = fmt.Sprintf("%f", gts.Value.(float32))

	case int64:
		value = strconv.FormatInt(gts.Value.(int64), 10)

	case int:
		value = strconv.Itoa(gts.Value.(int))

	case string:
		value = fmt.Sprintf("'%s'", url.QueryEscape(gts.Value.(string)))

	default:
		// Other types: just output their default format
		value = fmt.Sprintf("%v", gts.Value)
		value = url.QueryEscape(value)
	}

	b.WriteString(" ")
	b.WriteString(value)

	// Timestamp
	if gts.Ts > 0 {
		b.WriteString(" ")
		b.WriteString(strconv.FormatInt(gts.Ts, 10))
	}
	b.WriteString("\n")

	return b.Bytes()
}

func gtsFromMetric(name string, i interface{}, now int64, tags map[string]string) ([]*GTS, error) {
	m := []*GTS{}
	du := float64(time.Nanosecond)

	switch metric := i.(type) {

	case metrics.Counter:
		m = append(m, &GTS{Name: fmt.Sprintf("%s_count", name), Ts: now, Value: metric.Count(), Labels: tags})

	case metrics.Gauge:
		m = append(m, &GTS{Name: fmt.Sprintf("%s_value", name), Ts: now, Value: metric.Value(), Labels: tags})

	case metrics.GaugeFloat64:
		m = append(m, &GTS{Name: fmt.Sprintf("%s_value", name), Ts: now, Value: metric.Value(), Labels: tags})

	case metrics.Histogram:
		h := metric.Snapshot()
		ps := h.Percentiles([]float64{0.5, 0.75, 0.95, 0.99, 0.999})
		m = append(m, &GTS{Name: fmt.Sprintf("%s_count", name), Ts: now, Value: h.Count(), Labels: tags})
		m = append(m, &GTS{Name: fmt.Sprintf("%s_min", name), Ts: now, Value: h.Min(), Labels: tags})
		m = append(m, &GTS{Name: fmt.Sprintf("%s_max", name), Ts: now, Value: h.Max(), Labels: tags})
		m = append(m, &GTS{Name: fmt.Sprintf("%s_mean", name), Ts: now, Value: h.Mean(), Labels: tags})
		m = append(m, &GTS{Name: fmt.Sprintf("%s_std_dev", name), Ts: now, Value: h.StdDev(), Labels: tags})
		m = append(m, &GTS{Name: fmt.Sprintf("%s_50_percentile", name), Ts: now, Value: ps[0], Labels: tags})
		m = append(m, &GTS{Name: fmt.Sprintf("%s_75_percentile", name), Ts: now, Value: ps[1], Labels: tags})
		m = append(m, &GTS{Name: fmt.Sprintf("%s_95_percentile", name), Ts: now, Value: ps[2], Labels: tags})
		m = append(m, &GTS{Name: fmt.Sprintf("%s_99_percentile", name), Ts: now, Value: ps[3], Labels: tags})
		m = append(m, &GTS{Name: fmt.Sprintf("%s_999_percentile", name), Ts: now, Value: ps[4], Labels: tags})

	case metrics.Meter:
		meter := metric.Snapshot()
		m = append(m, &GTS{Name: fmt.Sprintf("%s_count", name), Ts: now, Value: meter.Count(), Labels: tags})
		m = append(m, &GTS{Name: fmt.Sprintf("%s_one_minute", name), Ts: now, Value: meter.Rate1(), Labels: tags})
		m = append(m, &GTS{Name: fmt.Sprintf("%s_five_minute", name), Ts: now, Value: meter.Rate5(), Labels: tags})
		m = append(m, &GTS{Name: fmt.Sprintf("%s_fifteen_minute", name), Ts: now, Value: meter.Rate15(), Labels: tags})
		m = append(m, &GTS{Name: fmt.Sprintf("%s_mean", name), Ts: now, Value: meter.RateMean(), Labels: tags})

	case metrics.Timer:
		t := metric.Snapshot()
		ps := t.Percentiles([]float64{0.5, 0.75, 0.95, 0.99, 0.999})
		m = append(m, &GTS{Name: fmt.Sprintf("%s_count", name), Ts: now, Value: t.Count(), Labels: tags})
		m = append(m, &GTS{Name: fmt.Sprintf("%s_min", name), Ts: now, Value: t.Min() / int64(du), Labels: tags})
		m = append(m, &GTS{Name: fmt.Sprintf("%s_max", name), Ts: now, Value: t.Max() / int64(du), Labels: tags})
		m = append(m, &GTS{Name: fmt.Sprintf("%s_mean", name), Ts: now, Value: t.Mean() / du, Labels: tags})
		m = append(m, &GTS{Name: fmt.Sprintf("%s_std_dev", name), Ts: now, Value: t.StdDev() / du, Labels: tags})
		m = append(m, &GTS{Name: fmt.Sprintf("%s_50_percentile", name), Ts: now, Value: ps[0] / du, Labels: tags})
		m = append(m, &GTS{Name: fmt.Sprintf("%s_75_percentile", name), Ts: now, Value: ps[1] / du, Labels: tags})
		m = append(m, &GTS{Name: fmt.Sprintf("%s_95_percentile", name), Ts: now, Value: ps[2] / du, Labels: tags})
		m = append(m, &GTS{Name: fmt.Sprintf("%s_99_percentile", name), Ts: now, Value: ps[3] / du, Labels: tags})
		m = append(m, &GTS{Name: fmt.Sprintf("%s_999_percentile", name), Ts: now, Value: ps[4] / du, Labels: tags})
		m = append(m, &GTS{Name: fmt.Sprintf("%s_one_minute", name), Ts: now, Value: t.Rate1(), Labels: tags})
		m = append(m, &GTS{Name: fmt.Sprintf("%s_five_minute", name), Ts: now, Value: t.Rate5(), Labels: tags})
		m = append(m, &GTS{Name: fmt.Sprintf("%s_fifteen_minute", name), Ts: now, Value: t.Rate15(), Labels: tags})
		m = append(m, &GTS{Name: fmt.Sprintf("%s_mean_rate", name), Ts: now, Value: t.RateMean(), Labels: tags})

	default:
		return nil, fmt.Errorf("Unknown metric type %T for metric '%s'", i, name)
	}

	return m, nil
}
