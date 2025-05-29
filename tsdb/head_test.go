package tsdb

import (
	"testing"
	"time"

	"github.com/pomyslowynick/scratcheus/labels"
)

var labelsLong labels.Labels

func Test_head_append(t *testing.T) {
	labelsLong = labels.Labels{
		labels.Label{Value: "prometheus_build_info", Name: "__name__"},
		labels.Label{Value: "release-2.54", Name: "branch"},
		labels.Label{Value: "amd64", Name: "goarch"},
		labels.Label{Value: "linux", Name: "goos"},
		labels.Label{Value: "go1.23.4", Name: "goversion"},
		labels.Label{Value: "c5e015d29534f06bd1d238c64a06b7ac41abdd7f", Name: "revision"},
		labels.Label{Value: "netgo,builtinassets,stringlabels", Name: "tags"},
		labels.Label{Value: "2.54.1", Name: "version"},
	}

	timestamp := uint64(time.Date(2025, time.April, 27, 12, 10, 10, 10, time.UTC).Unix())
	value := 2.75231
	head := NewHead()
	head.Append(labelsLong, timestamp, value)

	if memS := head.GetMemSeries(labelsLong); memS == nil {
		t.Errorf("memSeries wasn't correctly added to the head")
	} else {
		if len(memS.Bytes()) <= 2 {
			t.Errorf("No samples were appended")
		}
	}
}

func Test_head_getMemSeries(t *testing.T) {
	labelsLong = labels.Labels{
		labels.Label{Value: "prometheus_build_info", Name: "__name__"},
		labels.Label{Value: "release-2.54", Name: "branch"},
		labels.Label{Value: "amd64", Name: "goarch"},
		labels.Label{Value: "linux", Name: "goos"},
		labels.Label{Value: "go1.23.4", Name: "goversion"},
		labels.Label{Value: "c5e015d29534f06bd1d238c64a06b7ac41abdd7f", Name: "revision"},
		labels.Label{Value: "netgo,builtinassets,stringlabels", Name: "tags"},
		labels.Label{Value: "2.54.1", Name: "version"},
	}

	head := NewHead()
	memSeries := head.GetMemSeries(labelsLong)

	if memSeries != nil {
		t.Errorf("Should have returned a nil pointer")
	}

	timestamp := uint64(time.Date(2025, time.April, 27, 12, 10, 10, 10, time.UTC).Unix())
	value := 2.75231

	head.Append(labelsLong, timestamp, value)
	head.Append(labelsLong, timestamp, value)
	head.Append(labelsLong, timestamp, value)

	head.Append(labelsLong, timestamp+3, value+1)
	head.Append(labelsLong, timestamp+30, value+2)
	head.Append(labelsLong, timestamp+60, value+3)
	head.Append(labelsLong, timestamp+500, value+8)

	memSeries = head.GetMemSeries(labelsLong)
	if memSeries == nil {
		t.Errorf("Created series wasn't returned")
	}

	series := head.ReadMemSeries(labelsLong)
	expectedValues := []float64{2.75231, 2.75231, 2.75231, 3.75231, 4.75231, 5.75231, 10.75231}
	expectedTimestamps := []uint64{1745755810, 1745755810, 1745755810, 1745755813, 1745755840, 1745755870, 1745756310}

	for i, v := range series.samples {
		if expectedValues[i] != v.value {
			t.Errorf("Sample didn't contain expected timestamps: expected %v, actual %v", expectedValues[i], v.value)
		}

		if expectedTimestamps[i] != v.timestamp {
			t.Errorf("Sample didn't contain expected timestamps: expected %v, actual %v", expectedTimestamps[i], v.timestamp)
		}
	}
}
