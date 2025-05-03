package main

import (
	"fmt"
	"os"
	"time"

	"github.com/pomyslowynick/scratcheus/parser"
	"github.com/pomyslowynick/scratcheus/tsdb"
)

func main() {
	scrapeData, err := os.ReadFile("./test_files/metrics_five.txt")

	if err != nil {
		fmt.Println(err)
	}
	timestamp := uint64(time.Now().Unix())

	parsedEntries := parser.ParseScrapeData(scrapeData)

	newHead := tsdb.NewHead()

	for _, entry := range parsedEntries {
		newHead.Append(entry.Labels, timestamp, entry.Value)
	}

}
