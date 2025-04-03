package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/pomyslowynick/scratcheus/lexer"
	"github.com/pomyslowynick/scratcheus/storage"
)

type Metric struct {
	name       string
	timeseries map[string]*Timeseries
}

type Timeseries struct {
	labels map[string]string
	values []float64
}

func (t *Timeseries) appendValue(value float64) {
	t.values = append(t.values, value)
}

func main() {
	out, err := os.ReadFile("metrics_five.txt")

	if err != nil {
		fmt.Println(err)
	}
	parseScrapeData(out)
	time.Sleep(time.Millisecond * 500)

}

func parseScrapeData(scrapeData []byte) {
	newParser := lexer.NewParser(scrapeData)
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

		fmt.Println(et)
	}
}

func appendSeriesData() {
	a := storage.NewXorAppender()
	a.Append(time.Now().Unix(), 12.1)
}
