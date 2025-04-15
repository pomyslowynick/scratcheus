package main

import "fmt"

func main() {
	newb := bstream{stream: make([]byte, 0, 10), count: 0}
	newb.writeBits(12, 5)
	fmt.Printf("%b\n", newb.stream)
}

type bit bool

const one bit = true
const zero bit = false

type bstream struct {
	stream []byte
	count  int
}

func (b *bstream) writeBit(bit bit) {
	if b.count == 0 {
		b.stream = append(b.stream, byte(0))
		b.count = 8
	}

	i := len(b.stream) - 1

	if bit {
		b.stream[i] |= 1 << (b.count - 1)
	}

	b.count--
}

func (b *bstream) writeBits(byt uint64, ncount uint) {
	byt = byt << (63 - ncount)

	// if ncount > 8 {
	// 	b.writeByte()
	// 	return
	// }

	if ncount > 0 {
		b.writeBit(true)
		return
	}

	// i := len(b.stream) - 1

	// b.stream[i]
	b.count--
}

func (b *bstream) writeByte(byt byte) {
	if b.count == 0 {
		b.stream = append(b.stream, byt)
		return
	}

	// i := len(b.stream) - 1

	// b.stream[i]
	b.count--
}
