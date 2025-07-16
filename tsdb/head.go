package tsdb

import (
	"github.com/pomyslowynick/scratcheus/labels"
)

type memSeries struct {
	labels labels.Labels
	app    xorAppender
}

func newMemSeries(l labels.Labels) memSeries {
	return memSeries{
		labels: l,
		app:    NewAppender(),
	}
}

func (m *memSeries) Append(t uint64, v float64) {
	m.app.Append(t, v)
}

type Head struct {
	lastSeriesRef uint64
	series        map[uint64]*memSeries
	headChunk     Chunk
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

	if v, ok := h.series[id]; ok {
		reader := NewXorReader(v.app.b)
		return reader.readSeries()
	} else {
		return Series{}
	}
}

func (m *memSeries) Bytes() []byte {
	return m.app.Series()
}
