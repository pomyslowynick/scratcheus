package parser

import (
	"fmt"
	"os"
	"slices"
	"testing"

	"github.com/pomyslowynick/scratcheus/labels"
)

func Test_parser_metric(t *testing.T) {
	scrapeData, err := os.ReadFile("../test_files/metrics_five.txt")
	if err != nil {
		t.Fatalf("failed to read the metrics test file")
	}

	parsedSamples := ParseScrapeData(scrapeData)
	fmt.Println(parsedSamples)

	expectedSamples := map[string]ParsedSample{
		"prometheus_tp_requests_total": ParsedSample{
			Labels: labels.Labels{
				labels.Label{Name: "__name__", Value: "prometheus_http_requests_total"},
				labels.Label{Name: "code", Value: "200"},
				labels.Label{Name: "handler", Value: "/"},
			},
			Value: 0,
		},
		"prometheus_engine_query_log_enabled": ParsedSample{
			Labels: labels.Labels{
				labels.Label{Name: "__name__", Value: "prometheus_engine_query_log_enabled"},
			},
			Value: 0,
		},
		"prometheus_build_info": ParsedSample{
			Labels: labels.Labels{
				labels.Label{Name: "__name__", Value: "prometheus_build_info"},
				labels.Label{Name: "branch", Value: "release-2.54"},
				labels.Label{Name: "goarch", Value: "amd64"},
				labels.Label{Name: "goos", Value: "linux"},
				labels.Label{Name: "goversion", Value: "go1.23.4"},
				labels.Label{Name: "revision", Value: "c5e015d29534f06bd1d238c64a06b7ac41abdd7f"},
				labels.Label{Name: "tags", Value: "netgo,builtinassets,stringlabels"},
				labels.Label{Name: "version", Value: "2.54.1"},
			},
			Value: 1,
		},
		"process_cpu_seconds_total": ParsedSample{
			Labels: labels.Labels{
				labels.Label{Name: "__name__", Value: "process_cpu_seconds_total"},
			},
			Value: 0.47,
		},
		"process_max_fds": ParsedSample{
			Labels: labels.Labels{
				labels.Label{Name: "__name__", Value: "process_max_fds"},
			},
			Value: 1048576,
		},
	}

	for i, l := range parsedSamples {
		sample, ok := expectedSamples[i]
		if !ok {
			t.Fatalf("Sample not present in expected samples: %v", i)
		}

		if !slices.Equal(l.Labels, sample.Labels) {
			t.Fatalf("Sample labels not equal for %s: \ngot: %v \nexpected: %v", i, l.Labels, sample.Labels)
		}
	}
}
