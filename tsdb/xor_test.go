package tsdb

import (
	"slices"
	"testing"
	"time"
)

func Test_xor_append(t *testing.T) {
	appender := xorAppender{
		b: bstream{stream: make([]byte, 2)},
	}

	// 27th of April 2025, 12:10:10, 10ns, UTC
	// 1745755810
	date := time.Date(2025, time.April, 27, 12, 10, 10, 10, time.UTC)
	timestamp := uint64(date.Unix())
	value := 2.75231
	encodedSample := []byte{0, 1, 0, 0, 0, 0, 104, 14, 30, 162, 64, 6, 4, 187, 26, 243, 161, 77}

	appender.Append(timestamp, value)

	if appender.v != value {
		t.Errorf("Appended value wasn't registered")
	}

	if appender.t != timestamp {
		t.Errorf("Appended timestamp wasn't registered")
	}

	if !slices.Equal(appender.b.stream, encodedSample) {
		t.Errorf("Values and timestamp were not encoded as expected")
	}

	appender.Append(timestamp+60, value)

	if appender.ts_delta != 60 {
		t.Errorf("Delta between first and second sample timestamps should be 60: got instead %d", appender.ts_delta)
	}

	appender.Append(timestamp+90, value+1)

	if appender.ts_delta != 30 {
		t.Errorf("Delta between first and second sample timestamps should be 30: got instead %d", appender.ts_delta)
	}
}

func Test_xor_read(t *testing.T) {
	appender := xorAppender{
		b: bstream{stream: make([]byte, 2)},
	}

	// 27th of April 2025, 12:10:10, 10ns, UTC
	// 1745755810
	date := time.Date(2025, time.April, 27, 12, 10, 10, 10, time.UTC)
	timestamp := uint64(date.Unix())
	value := 2.75231

	appender.Append(timestamp, value)

	if appender.v != value {
		t.Errorf("Appended value wasn't registered")
	}

	appender.Append(timestamp+30, value+1)

	appender.Append(timestamp+60, value+2)

	reader := NewXorReader(appender.b)
	retrievedSeries := reader.readSeries()

	for i, v := range retrievedSeries.samples {
		if v.value != value+float64(i) {
			t.Errorf("Value %f not equal to expected value of %f", v.value, value+float64(i))
		}
	}

	for i, ts := range retrievedSeries.samples {
		if ts.timestamp != timestamp+uint64(i*30) {
			t.Errorf("Timestamp %d not equal to expected value of %d", ts.timestamp, timestamp+uint64(i*30))
		}
	}
}
