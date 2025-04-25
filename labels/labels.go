package labels

type Labels []Label

type Label struct {
	Value string
	Name  string
}

type Builder struct {
	add Labels
}

func (b *Builder) Add(name, value string) {

}

func (l *Label) Bytes() []byte {
	var labelsAsBytes []byte

	for _, b := range []byte(l.Name) {
		labelsAsBytes = append(labelsAsBytes, b)
	}

	for _, b := range []byte(l.Value) {
		labelsAsBytes = append(labelsAsBytes, b)
	}
	return labelsAsBytes
}
