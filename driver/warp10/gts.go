package warp10

import (
	"fmt"
	"math"
	"net/url"
)

// GTS struct
type GTS struct {
	Ts     float64
	Name   string
	Labels map[string]string
	Value  interface{}
}

// Encode a GTS to the Sensision format
// TS/LAT:LON/ELEV NAME{LABELS} VALUE
func (gts *GTS) Encode() []byte {
	sensision := ""

	// Timestamp
	if !math.IsNaN(gts.Ts) {
		sensision += fmt.Sprintf("%d", int(gts.Ts))
	}

	// Class
	sensision += fmt.Sprintf("// %s{", url.QueryEscape(gts.Name))

	sep := ""
	for k, v := range gts.Labels {
		sensision += sep + url.QueryEscape(k) + "=" + url.QueryEscape(v)
		sep = ","
	}
	sensision += "} "

	// value
	switch gts.Value.(type) {
	case bool:
		if gts.Value.(bool) {
			sensision += "T"
		} else {
			sensision += "F"
		}

	case float64:
		sensision += fmt.Sprintf("%f", gts.Value.(float64))

	case int64:
		sensision += fmt.Sprintf("%d", gts.Value.(int64))

	case float32:
		sensision += fmt.Sprintf("%f", gts.Value.(float32))

	case int:
		sensision += fmt.Sprintf("%d", gts.Value.(int))

	case string:
		sensision += fmt.Sprintf("'%s'", url.QueryEscape(gts.Value.(string)))

	default:
		// Other types: just output their default format
		strVal := fmt.Sprintf("%v", gts.Value)
		sensision += url.QueryEscape(strVal)
	}
	sensision += "\r\n"

	return []byte(sensision)
}
