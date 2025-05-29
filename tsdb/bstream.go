// Copyright (c) 2015,2016 Damian Gryski <damian@gryski.com>
// All rights reserved.

// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are met:

// * Redistributions of source code must retain the above copyright notice,
// this list of conditions and the following disclaimer.
//
// * Redistributions in binary form must reproduce the above copyright notice,
// this list of conditions and the following disclaimer in the documentation
// and/or other materials provided with the distribution.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
// ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
// WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE
// FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
// DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR
// SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER
// CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY,
// OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
// OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
package tsdb

type bit bool

const (
	one  = true
	zero = false
)

type bstream struct {
	stream []byte
	count  int
}

func (b *bstream) writeBit(bit bit) {
	if b.count == 0 {
		b.stream = append(b.stream, byte(0))
		b.count += 8
	}

	i := len(b.stream) - 1

	if bit {
		b.stream[i] |= byte(1 << (b.count - 1))

	}

	b.count--
}

func (b *bstream) writeByte(byt byte) {
	if b.count == 0 {
		b.stream = append(b.stream, byt)
		return
	}

	i := len(b.stream) - 1

	b.stream[i] |= byt >> (8 - b.count)

	b.stream = append(b.stream, byt<<b.count)
}

func (b *bstream) writeBits(bits uint64, nbits int) {
	bits = bits << (64 - nbits)

	for nbits >= 8 {
		byt := byte(bits >> 56)
		b.writeByte(byt)
		bits <<= 8
		nbits -= 8
	}

	for nbits > 0 {
		bi := bit(bits>>63 == 1)
		b.writeBit(bi)
		bits <<= 1
		nbits--
	}
}

func (b *bstream) bytes() []byte {
	return b.stream
}

func (b *bstream) Reset(stream []byte) {
	b.stream = stream
	b.count = 0
}

type Iterator struct {
	b         bstream
	countBit  int
	countByte int
}

func NewIterator(b bstream) Iterator {
	return Iterator{
		b:        b,
		countBit: 0,
	}
}

func (i *Iterator) nextBit() (ret bit) {
	// We should probably return an error if nextBit is called when stream has been iterated over
	if len(i.b.stream)-1 < i.countByte {
		panic("out of bounds")
	}

	tempByte := (i.b.stream[i.countByte] >> (7 - i.countBit)) & byte(1)
	i.countBit++

	if i.countBit == 8 {
		i.countBit = 0
		i.countByte++
	}

	switch tempByte {
	case 0:
		return zero
	case 1:
		return one
	default:
		panic("woooot, you got bit which wasn't a 1 or 0, you are running a quantum compute")
	}
}

func (i *Iterator) nextBits(count int) (ret []bit) {
	// We should probably return an error if nextBit is called when stream has been iterated over
	if len(i.b.stream)-1 < i.countByte {
		panic("out of bounds")
	}

	for y := 0; count > y; y++ {
		b := i.nextBit()
		ret = append(ret, b)
	}

	return ret
}

func (i *Iterator) nextByte() (ret byte) {
	// We should probably return an error if nextBit is called when stream has been iterated over
	if len(i.b.stream)-1 < i.countByte {
		panic("out of bounds")
	}

	ret = i.b.stream[i.countByte]
	i.countByte++
	i.countBit = 0

	return ret
}

func (i *Iterator) nextBytes(count int) (ret []byte) {
	if len(i.b.stream)-1 < i.countByte {
		panic("out of bounds")
	}

	for y := 0; count > y; y++ {
		ret = append(ret, i.nextByte())
	}

	return ret
}
