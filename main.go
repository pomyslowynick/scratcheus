package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/pomyslowynick/scratcheus/labels"
	"github.com/pomyslowynick/scratcheus/lexer"
	"github.com/pomyslowynick/scratcheus/tsdb"
)

func main() {
	scrapeData, err := os.ReadFile("./test_files/metrics_five.txt")

	if err != nil {
		fmt.Println(err)
	}
	timestamp := uint64(time.Now().Unix())

	parsedEntries := parseScrapeData(scrapeData)

	newHead := tsdb.NewHead()

	for _, entry := range parsedEntries {
		newHead.Append(entry.labels, timestamp, entry.value)
	}

}

type ParsedSample struct {
	labels labels.Labels
	value  float64
}

func parseScrapeData(scrapeData []byte) map[string]ParsedSample {
	newParser := lexer.NewParser(scrapeData)

	seriesSamples := make(map[string]ParsedSample)
	for {
		var (
			et  lexer.Entry
			err error
		)

		if et, err = newParser.Next(); err != nil {
			if errors.Is(err, io.EOF) {
				err = nil
			}
			break
		}

		if et == lexer.EntrySeries {
			_, _, value := newParser.Series()
			metricName, labels := newParser.Labels()
			seriesSamples[metricName] = ParsedSample{
				value:  value,
				labels: labels,
			}
		}
	}

	return seriesSamples
}
