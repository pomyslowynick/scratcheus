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

	timestamp := uint64(time.Now().Unix())
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

	timestamp := uint64(time.Now().Unix())
	value := 2.75231
	head.Append(labelsLong, timestamp, value)

	memSeries = head.GetMemSeries(labelsLong)

	if memSeries == nil {
		t.Errorf("Created series wasn't returned")
	}
}
