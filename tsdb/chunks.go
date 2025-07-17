package tsdb

import (
	"time"
)

type Chunk struct {
	ts       int64
	mmaped   bool
	app      xorAppender
	previous *Chunk
}

func cutNewChunk() *Chunk {
	return &Chunk{
		app:    NewAppender(),
		ts:     time.Now().Unix(),
		mmaped: false,
	}
}

func (c *Chunk) Append(t uint64, v float64) {
	c.app.Append(t, v)
}

func (c *Chunk) Bytes() bstream {
	return c.app.b
}

func (c *Chunk) SamplesNum() int {
	return c.app.SamplesNum()
}

func (c *Chunk) chunksListLength() int {
	chunk := c
	counter := 1

	for chunk.previous != nil {
		counter++
		chunk = chunk.previous
	}

	return counter
}
