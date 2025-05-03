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
	date := time.Date(2025, time.April, 27, 12, 10, 10, 10, time.UTC)
	timestamp := uint64(date.Unix())
	value := 2.75231
	encodedSample := []byte{0, 1, 0, 0, 0, 0, 104, 14, 30, 162, 64, 6, 4, 187, 26, 243, 161, 77}

	appender.Append(timestamp, value)

	// After append: [0,1,0,0,0,0,104,14,30,162,64,6,4,187,26,243,161,77]
	if appender.v != value {
		t.Errorf("Appended value wasn't registered")
	}

	if appender.t != timestamp {
		t.Errorf("Appended timestamp wasn't registered")
	}

	if !slices.Equal(appender.b.stream, encodedSample) {
		t.Errorf("Values and timestamp were not encoded as expected")
	}

	appender.Append(timestamp, value)
	// After [0,2,0,0,0,0,104,14,30,162,64,6,4,187,26,243,161,77,0,0,0,0,0,0,0,0,0]

	if appender.ts_delta != 0 {
		t.Errorf("Delta between first and second sample timestamps should be 0")
	}

	if appender.b.stream[len(appender.b.stream)-1] != 0 {

		t.Errorf("Delta between first and second sample values should be 0")
	}

	appender.Append(timestamp+30, value+1)
	// After [0,3,0,0,0,0,104,14,30,162,64,6,4,187,26,243,161,77,0,0,0,0,0,0,0,0,79,108,6]

	if appender.ts_delta != 30 {
		t.Errorf("Delta between second and third sample timestamps should be 30")
	}
}
