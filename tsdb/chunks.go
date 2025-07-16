package tsdb

import (
	"time"
)

type Chunk struct {
	series     *memSeries
	samplesNum int8
	ts         int64
	mmaped     bool
	app        xorAppender
}

func cutNewChunk() Chunk {
	return Chunk{
		samplesNum: 0,
		ts:         time.Now().Unix(),
		mmaped:     false,
	}
}

func (c *Chunk) Append(t uint64, v float64) {
	c.app.Append(t, v)
}
