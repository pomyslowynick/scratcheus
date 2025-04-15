package main

import (
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
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
	b := []byte{1, 1, 5, 4, 300}
	fmt.Println(binary.BigEndian.Uint16(b))
	fmt.Println(uint16(b[0]) << 8)
}

// My naive attempt at parsing scrape data, keeping it to include it in the tutorial as an anti-example
func scrape(scrapeData []byte) {
	allMetrics := make(map[string]Metric, 0)
	outAsStr := string(scrapeData)

	for _, item := range strings.Split(outAsStr, "\n") {
		if item == "" {
			continue
		}

		splitMetric := strings.Split(item, " ")

		nameAndLabels := splitMetric[0]
		value := splitMetric[1]
		convertedValue, err := strconv.ParseFloat(value, 64)

		if err != nil {
			panic("Couldn't convert string to Float")
		}

		splitNameAndLabels := strings.Split(nameAndLabels, "{")
		metricName := splitNameAndLabels[0]

		if _, ok := allMetrics[metricName]; !ok {
			// labels
			parsedLabels := make(map[string]string)
			if len(splitNameAndLabels) > 1 {
				for _, label := range strings.Split(splitNameAndLabels[1], ",") {
					splitLabel := strings.Split(label, "=")
					labelValue := strings.Join(splitLabel[1:], "")
					parsedLabels[splitLabel[0]] = labelValue

				}

			}

			// values
			valuesArray := []float64{convertedValue}
			newM := Metric{
				name:       metricName,
				timeseries: make(map[string]*Timeseries),
			}
			newM.timeseries[nameAndLabels] = &Timeseries{
				labels: parsedLabels,
				values: valuesArray,
			}
			allMetrics[metricName] = newM
		} else {
			if val, ok := allMetrics[metricName].timeseries[nameAndLabels]; ok {
				val.appendValue(convertedValue)
			} else {
				parsedLabels := make(map[string]string)
				if len(splitNameAndLabels) > 1 {
					for _, label := range strings.Split(splitNameAndLabels[1], ",") {
						splitLabel := strings.Split(label, "=")
						labelValue := strings.Join(splitLabel[1:], "")
						parsedLabels[splitLabel[0]] = labelValue
					}

				}

				// values
				valuesArray := []float64{convertedValue}
				allMetrics[metricName].timeseries[nameAndLabels] = &Timeseries{
					labels: parsedLabels,
					values: valuesArray,
				}
			}
		}
	}
	fmt.Println(allMetrics)
}
