package tsdb

import (
	"github.com/pomyslowynick/scratcheus/labels"
)

type Head struct {
	lastSeriesRef uint64
	series        map[uint64]*memSeries
}

func NewHead() Head {
	return Head{
		lastSeriesRef: 1,
		series:        make(map[uint64]*memSeries),
	}
}

func (h *Head) createOrGetMemSeries(l labels.Labels) *memSeries {
	id, err := l.HashLabels()

	if err != nil {
		panic("Should never happen")
	}

	if v, ok := h.series[id]; ok {
		return v
	} else {
		newSeries := newMemSeries(l)
		h.series[id] = &newSeries
		return &newSeries
	}
}

func (h *Head) Append(l labels.Labels, t uint64, v float64) {
	memSeries := h.createOrGetMemSeries(l)

	memSeries.Append(t, v)
}

func (h *Head) GetMemSeries(l labels.Labels) *memSeries {
	id, err := l.HashLabels()
	if err != nil {
		panic("Should never happen")
	}

	if v, ok := h.series[id]; ok {
		return v
	} else {
		return nil
	}
}

func (h *Head) ReadMemSeries(l labels.Labels) Series {
	id, err := l.HashLabels()
	if err != nil {
		panic("Should never happen")
	}

	if series, ok := h.series[id]; ok {
		reader := NewXorReader(series.headChunk.Bytes())
		return reader.readSeries()
	} else {
		return Series{}
	}
}

type memSeries struct {
	labels    labels.Labels
	headChunk *Chunk
}

func newMemSeries(l labels.Labels) memSeries {
	return memSeries{
		labels:    l,
		headChunk: cutNewChunk(),
	}
}

func (m *memSeries) Append(t uint64, v float64) {
	if m.headChunk.SamplesNum() >= 120 {
		previous := m.headChunk
		m.headChunk = cutNewChunk()
		m.headChunk.previous = previous
	}
	m.headChunk.Append(t, v)
}

func (m *memSeries) headChunkBytes() []byte {
	return m.headChunk.app.Series()
}
