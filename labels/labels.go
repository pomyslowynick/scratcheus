package labels

import "hash/fnv"

type Labels []Label

type Label struct {
	Name  string
	Value string
}

type Builder struct {
	add Labels
}

func (b *Builder) Add(name, value string) {

}

func (l *Label) Bytes() []byte {
	var labelsAsBytes []byte

	labelsAsBytes = append(labelsAsBytes, []byte(l.Name)...)

	labelsAsBytes = append(labelsAsBytes, []byte(l.Value)...)
	return labelsAsBytes
}

func (l Labels) HashLabels() (uint64, error) {
	newHash := fnv.New64()

	for _, v := range l {
		_, err := newHash.Write(v.Bytes())
		if err != nil {
			return 0, err
		}

	}
	return newHash.Sum64(), nil
}
