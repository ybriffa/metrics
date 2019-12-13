package metrics

import (
	"errors"
	"reflect"
	"strconv"
	"strings"

	"github.com/rcrowley/go-metrics"
	log "github.com/sirupsen/logrus"
)

var (
	ErrNotPointer                error = errors.New("structure given is not a pointer")
	ErrNotInterface              error = errors.New("field given is not an interface")
	ErrMetricsTypeUnhandled      error = errors.New("type of metric not handled")
	ErrMetricsNameDuplicated     error = errors.New("metric name duplicated in tags (or var name)")
	ErrUnknownSampleType         error = errors.New("unknown sample type")
	ErrInvalidUniformSampleValue error = errors.New("invalid uniform value sample")
	ErrInvalidExpSampleFormat    error = errors.New("invalid exp sample value format")
	ErrInvalidExpSampleValue     error = errors.New("invalid exp sample value")
)

func sanitize(name string) string {
	return strings.ToLower(name)
}

// RegistryFromStruct takes a data structure and creates a registry from its fields.
func RegistryFromStruct(s interface{}) (metrics.Registry, error) {
	v := reflect.ValueOf(s)

	if v.Kind() != reflect.Ptr {
		return nil, ErrNotPointer
	}

	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	t := v.Type()
	ret := metrics.NewRegistry()
	names := map[string]struct{}{}
	for idx := 0; idx < t.NumField(); idx++ {
		// Getting information about the field
		field := t.Field(idx)
		fieldValue := v.Field(idx)

		// Checking if the variable is settable, otherwise does not interest us
		if !fieldValue.CanSet() {
			log.Debugf("[metrics] cannot set field %s, skipping ", field.Name)
			continue
		}

		// Getting the name to register, in the tag `metrics:""` or the name of the field
		name := field.Name
		if tmp := field.Tag.Get("metrics"); tmp != "" {
			name = tmp
		}
		name = sanitize(name)
		if _, exists := names[name]; exists {
			return nil, ErrMetricsNameDuplicated
		}
		names[name] = struct{}{}

		// Instantiate the correct type or use the settled one, and push it in the struct
		var newVar interface{}
		if !fieldValue.IsNil() {
			newVar = fieldValue.Interface()
		} else {
			var err error
			newVar, err = metricFromField(fieldValue, field.Tag)
			if err != nil {
				log.Debugf("[metrics] unabled to instanciate metrics from field %s : %s", field.Name, err)
				continue
			}
			fieldValue.Set(reflect.ValueOf(newVar).Convert(fieldValue.Type()))
		}

		// Add it in the registry
		ret.Register(name, newVar)
	}
	return ret, nil
}

func metricFromField(v reflect.Value, tag reflect.StructTag) (interface{}, error) {
	if !v.CanInterface() {
		return nil, ErrNotInterface
	}

	switch v.Type().String() {

	case "metrics.Counter":
		return metrics.NewCounter(), nil

	case "metrics.Gauge":
		return metrics.NewGauge(), nil

	case "metrics.GaugeFloat64":
		return metrics.NewGaugeFloat64(), nil

	case "metrics.Meter":
		return metrics.NewMeter(), nil

	case "metrics.Timer":
		return metrics.NewTimer(), nil

	case "metrics.Histogram":
		return newHistogram(tag.Get("metrics_sample"), tag.Get("metrics_sample_value"))
	}

	return nil, ErrMetricsTypeUnhandled
}

func newHistogram(sampleType, sampleValue string) (metrics.Histogram, error) {
	var s metrics.Sample

	switch sampleType {
	case "exp":
		splitted := strings.Split(sampleValue, "-")
		if len(splitted) != 2 {
			return nil, ErrInvalidExpSampleFormat
		}

		reservoirSize, err := strconv.Atoi(splitted[0])
		if err != nil {
			return nil, ErrInvalidExpSampleValue
		}

		alpha, err := strconv.ParseFloat(splitted[1], 64)
		if err != nil {
			return nil, ErrInvalidExpSampleValue
		}

		s = metrics.NewExpDecaySample(reservoirSize, alpha)
	case "", "uniform":
		reservoirSize := 999
		if sampleValue != "" {
			var err error
			reservoirSize, err = strconv.Atoi(sampleValue)
			if err != nil {
				return nil, ErrInvalidUniformSampleValue
			}
		}
		s = metrics.NewUniformSample(reservoirSize)
	default:
		return nil, ErrUnknownSampleType
	}

	return metrics.NewHistogram(s), nil
}
