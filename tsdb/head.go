package tsdb

import (
	"hash/fnv"

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
	series        map[uint64]memSeries
	seriesHashes  map[string]uint64
}

func NewHead() Head {
	return Head{
		lastSeriesRef: 1,
		series:        make(map[uint64]memSeries),
	}
}

func (h *Head) hashLabels(l labels.Labels) uint64 {
	newHash := fnv.New64()

	for _, v := range l {
		_, _ = newHash.Write(v.Bytes())

	}
	return newHash.Sum64()

}

func (h *Head) createOrGetMemSeries(l labels.Labels) *memSeries {
	id := h.hashLabels(l)

	if id == 0 {
		panic("Should never happen")
	}

	if v, ok := h.series[id]; ok {
		return &v
	} else {
		newSeries := newMemSeries(l)
		h.series[id] = newSeries
		return &newSeries
	}
}

func (h *Head) Append(labels labels.Labels, t uint64, v float64) {
	memSeries := h.createOrGetMemSeries(labels)

	memSeries.Append(t, v)
}
