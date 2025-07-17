package tsdb

import (
	"testing"
	"time"

	"github.com/pomyslowynick/scratcheus/labels"
)

var labelsLong labels.Labels = labels.Labels{
	labels.Label{Value: "prometheus_build_info", Name: "__name__"},
	labels.Label{Value: "release-2.54", Name: "branch"},
	labels.Label{Value: "amd64", Name: "goarch"},
	labels.Label{Value: "linux", Name: "goos"},
	labels.Label{Value: "go1.23.4", Name: "goversion"},
	labels.Label{Value: "c5e015d29534f06bd1d238c64a06b7ac41abdd7f", Name: "revision"},
	labels.Label{Value: "netgo,builtinassets,stringlabels", Name: "tags"},
	labels.Label{Value: "2.54.1", Name: "version"},
}

var timestamp uint64 = uint64(time.Date(2025, time.April, 27, 12, 10, 10, 10, time.UTC).Unix())

func Test_head_append(t *testing.T) {
	value := 2.75231
	head := NewHead()
	head.Append(labelsLong, timestamp, value)

	if memS := head.GetMemSeries(labelsLong); memS == nil {
		t.Errorf("memSeries is nil after head.Append")
	} else {
		if len(memS.headChunkBytes()) <= 2 {
			t.Errorf("No samples were appended")
		}
	}
}

func Test_head_getMemSeries(t *testing.T) {
	head := NewHead()
	memSeries := head.GetMemSeries(labelsLong)

	if memSeries != nil {
		t.Errorf("Memseries was returned for labels which shouldn't belong to any")
	}

	value := 2.75231

	for _ = range 3 {
		head.Append(labelsLong, timestamp, value)
	}

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
			t.Errorf("Sample didn't contain expected value: expected %v, actual %v", expectedValues[i], v.value)
		}

		if expectedTimestamps[i] != v.timestamp {
			t.Errorf("Sample didn't contain expected timestamp: expected %v, actual %v", expectedTimestamps[i], v.timestamp)
		}
	}
}

func Test_head_cutNewChunk(t *testing.T) {
	head := NewHead()

	value := 2.75231

	for _ = range 121 {
		head.Append(labelsLong, timestamp, value)
	}

	memSeries := head.GetMemSeries(labelsLong)
	if memSeries == nil {
		t.Errorf("Created series wasn't returned")
	}

	if memSeries.headChunk.previous == nil {
		t.Errorf("No new chunk created")
	}

	for _ = range 121 {
		head.Append(labelsLong, timestamp, value)
	}

	if memSeries.headChunk.chunksListLength() != 3 {
		t.Errorf("Head chunks list should be equal to 3, instead it's: %d", memSeries.headChunk.chunksListLength())
	}
}
