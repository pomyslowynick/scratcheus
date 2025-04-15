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
