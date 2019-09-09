package metrics

import (
	"testing"

	"github.com/rcrowley/go-metrics"
)

func TestRegistryFromStruct(t *testing.T) {
	// 0 struct not a pointer
	var notAPoint struct{}

	_, err := RegistryFromStruct(notAPoint)
	if err == nil {
		t.Fatal("Test #0 failed : did not return an error")
	}

	// 1 same tag defined
	var multitpleTags struct {
		Toto metrics.Counter `metrics:"my-tag"`
		Titi metrics.Counter `metrics:"my-tag"`
	}

	_, err = RegistryFromStruct(&multitpleTags)
	if err != ErrMetricsNameDuplicated {
		t.Fatalf("Test #1 failed : expected error `%s` and got `%s`", ErrMetricsNameDuplicated, err)
	}

	// 2 tag and field name collision
	var tagAndFieldCollision struct {
		Toto metrics.Counter `metrics:"titi"`
		Titi metrics.Counter
	}

	_, err = RegistryFromStruct(&tagAndFieldCollision)
	if err != ErrMetricsNameDuplicated {
		t.Fatalf("Test #2 failed : expected error `%s` and got `%s`", ErrMetricsNameDuplicated, err)
	}

	// 3 correct
	var correct struct {
		Toto metrics.Counter
		Titi metrics.Meter
		Tutu metrics.Histogram `metrics_sample_value:"42"`
	}

	r, err := RegistryFromStruct(&correct)
	if err != nil {
		t.Fatalf("Test #3 failed : error is not nil : %s", err)
	}

	c := r.Get("toto")
	_, ok := c.(metrics.Counter)
	if !ok {
		t.Fatal("Test #3 failed : field toto is not type of Counter")
	}

	m := r.Get("titi")
	_, ok = m.(metrics.Meter)
	if !ok {
		t.Fatal("Test #3 failed : field titi is not type of Meter")
	}
}

func TestNewHistogram(t *testing.T) {
	tests := []struct {
		sampleType, sampleValue string
		errExpected             error
	}{
		//0 invalid type
		{
			sampleType:  "uknz",
			sampleValue: "",
			errExpected: ErrUnknownSampleType,
		},

		//1 not 2 values
		{
			sampleType:  "exp",
			sampleValue: "456446",
			errExpected: ErrInvalidExpSampleFormat,
		},

		//2 not int
		{
			sampleType:  "exp",
			sampleValue: "abc-12.5",
			errExpected: ErrInvalidExpSampleValue,
		},

		//3 not float
		{
			sampleType:  "exp",
			sampleValue: "42-abc",
			errExpected: ErrInvalidExpSampleValue,
		},

		//4
		{
			sampleType:  "exp",
			sampleValue: "42-12.5",
			errExpected: nil,
		},

		//5 not int
		{
			sampleType:  "",
			sampleValue: "abc",
			errExpected: ErrInvalidUniformSampleValue,
		},

		//6 not int with real name
		{
			sampleType:  "uniform",
			sampleValue: "abc",
			errExpected: ErrInvalidUniformSampleValue,
		},

		//7 not int with real name
		{
			sampleType:  "uniform",
			sampleValue: "42",
			errExpected: nil,
		},
	}

	for i, test := range tests {
		_, err := newHistogram(test.sampleType, test.sampleValue)
		if err != test.errExpected {
			t.Fatalf("test #%d failed : expected `%s` and got `%s`", i, test.errExpected, err)
		}
	}
}
