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

import (
	"encoding/binary"
	"fmt"
	"math"
	"math/bits"
)

type Chunk struct {
	b bstream
}

type xorAppender struct {
	b              bstream
	t              uint64
	v              float64
	leading_zeros  int
	trailing_zeros int
	ts_delta       uint64
}

func (x *xorAppender) Append(t uint64, v float64) {
	num := binary.BigEndian.Uint16(x.b.stream)

	switch num {
	case 0:
		x.b.writeBits(t, 64)
		x.b.writeBits(math.Float64bits(v), 64)
	case 1:
		ts_delta := t - x.t
		x.b.writeBits(ts_delta, 64)
		x.ts_delta = ts_delta

		x.writeVDelta(v)
	default:
		ts_delta := t - x.t
		dod := int64(ts_delta - x.ts_delta)
		x.ts_delta = ts_delta
		switch {
		case dod == 0:
			x.b.writeBit(zero)
		case bitsRange(dod, 7):
			x.b.writeBits(0b10, 2)
			x.b.writeBits(uint64(dod), 6)
		case bitsRange(dod, 9):
			x.b.writeBits(0b110, 3)
			x.b.writeBits(uint64(dod), 5)
		case bitsRange(dod, 12):
			x.b.writeBits(0b1110, 4)
			x.b.writeBits(uint64(dod), 5)
		default:
			x.b.writeBits(0b1111, 4)
			x.b.writeBits(uint64(dod), 32)
		}
		x.writeVDelta(v)
	}

	num += 1
	x.b.stream[0] = byte(num >> 8)
	x.b.stream[1] = byte(num)

	x.t = t
	x.v = v
}

func (x *xorAppender) writeVDelta(v float64) {
	delta := math.Float64bits(v) ^ math.Float64bits(x.v)
	if delta == 0 {
		x.b.writeBit(zero)
		return
	}

	x.b.writeBit(one)

	leading_zeros := bits.LeadingZeros64(delta)
	trailing_zeros := bits.TrailingZeros64(delta)

	if leading_zeros == x.leading_zeros && trailing_zeros == x.trailing_zeros {
		x.b.writeBit(zero)
		x.b.writeBits(delta, 64-(leading_zeros+trailing_zeros))
		return
	}

	x.b.writeBit(one)
	x.b.writeBits(uint64(leading_zeros), 5)
	x.b.writeBits(uint64(64-(leading_zeros+trailing_zeros)), 6)
	x.b.writeBits(delta>>trailing_zeros, 64-(leading_zeros+trailing_zeros))

	x.leading_zeros = leading_zeros
	x.trailing_zeros = trailing_zeros
}

func (x *xorAppender) Series() []byte {
	return x.b.stream
}

type xorReader struct {
	stream bstream

	timestamp     uint64
	ts_delta      uint64
	value         float64
	tempValue     uint64
	leadingZeros  int
	trailingZeros int
}

type Series struct {
	values     []float64
	timestamps []uint64
}

func NewXorReader(s bstream) xorReader {
	return xorReader{stream: s}
}

// Refactor this function, turn it into smaller functions
func (x *xorReader) ReadSeries() Series {
	var values []float64
	var timestamps []uint64

	streamIterator := NewIterator(x.stream)

	firstByte := streamIterator.nextByte()
	secondByte := streamIterator.nextByte()

	samplesNum := (int16(firstByte) << 8) + int16(secondByte)

	if samplesNum == 0 {
		return Series{}
	}

	// Read timestamp - no encoding
	timestampBytes := streamIterator.nextBytes(8)

	for i, byt := range timestampBytes {
		x.timestamp += uint64(byt) << ((7 - i) * 8)
	}
	timestamps = append(timestamps, x.timestamp)

	// Read first float value - no encoding
	valueBytes := streamIterator.nextBytes(8)

	for i, byt := range valueBytes {
		x.tempValue += uint64(byt) << ((7 - i) * 8)
	}
	x.leadingZeros = bits.LeadingZeros64(x.tempValue)
	x.trailingZeros = bits.TrailingZeros64(x.tempValue)
	value := math.Float64frombits(x.tempValue)

	x.value = value
	values = append(values, value)

	if samplesNum == 1 {
		return Series{
			values:     values,
			timestamps: timestamps,
		}
	}

	// Second tuple is always the first timestamp delta and xored value
	timestampDelta := streamIterator.nextBytes(8)

	for i, byt := range timestampDelta {
		x.ts_delta += uint64(byt) << ((7 - i) * 8)
	}
	timestamps = append(timestamps, x.timestamp+x.ts_delta)
	x.timestamp = x.timestamp + x.ts_delta

	// Second value
	secondValue := x.readXorEncodedValue(&streamIterator)
	values = append(values, secondValue)
	x.value = secondValue

	// Read rest of the encoded samples
	//   Timestamps after first delta are deltas of deltas with variable encoding length
	//   TODO: Move it to a separate function
	var tsDod uint64
	tsDodFirstBit := streamIterator.nextBit()
	switch tsDodFirstBit {
	case zero:
		tsDod = 0
	case one:
		tsDodSecondBit := streamIterator.nextBit()
		switch tsDodSecondBit {
		case zero:
			tempDod := streamIterator.nextBits(7)
			tsDod = bitsSliceToInt(tempDod)
		case one:
			tsDodThirdBit := streamIterator.nextBit()
			switch tsDodThirdBit {
			case zero:
				tempDod := streamIterator.nextBits(9)
				tsDod = bitsSliceToInt(tempDod)
			case one:
				tsDodFourthBit := streamIterator.nextBit()
				switch tsDodFourthBit {
				case zero:
					tempDod := streamIterator.nextBits(12)
					tsDod = bitsSliceToInt(tempDod)
				case one:
					tempDod := streamIterator.nextBits(32)
					tsDod = bitsSliceToInt(tempDod)

				}
			}
		}

	}

	timestamps = append(timestamps, x.timestamp+x.ts_delta+tsDod)
	x.timestamp = x.timestamp + x.ts_delta + tsDod
	x.ts_delta = x.ts_delta + tsDod

	// Second value
	values = append(values, x.readXorEncodedValue(&streamIterator))

	return Series{
		values:     values,
		timestamps: timestamps,
	}
}

func (x *xorReader) readXorEncodedValue(si *Iterator) float64 {
	var finalValue float64
	isDeltaZero := !bool(si.nextBit())

	// TODO: Double switch is a smell, there might be a better idiom for that
	switch {
	case isDeltaZero:
		return finalValue
	case !isDeltaZero:
		controlBit := bool(si.nextBit())
		switch {
		case controlBit:
			leadingZeros := bitsSliceToInt(si.nextBits(5))
			valueLen := bitsSliceToInt(si.nextBits(6))
			xoredValue := bitsSliceToInt(si.nextBits(int(valueLen)))
			trailingZeros := 64 - (leadingZeros + valueLen)
			decodedValue := math.Float64bits(x.value) ^ (xoredValue << trailingZeros)
			dvFloat := math.Float64frombits(decodedValue)
			x.leadingZeros = int(leadingZeros)
			x.trailingZeros = int(trailingZeros)

			return dvFloat

		case !controlBit:
			sigBits := 64 - (x.leadingZeros + x.trailingZeros)
			xoredValue := si.nextBits(sigBits)

			valueBits := make([]bit, x.leadingZeros)

			xoredShiftedValue := bitsSliceToInt(xoredValue)

			for _, v := range xoredValue {
				valueBits = append(valueBits, v)
			}
			fmt.Printf("Decoded value without the control bit %b\n", xoredShiftedValue)
		}

	}
	// Should never reach this part
	return finalValue
}

func bitsRange(v int64, nbits int) bool {
	return 1<<(nbits-1) >= v && -((1<<(nbits-1))-1) <= v
}

func bitsSliceToInt(bits []bit) uint64 {
	var bitsAsInt uint64

	for i, b := range bits {
		if b {
			bitsAsInt += 1 << ((len(bits) - 1) - i)
		}
	}

	return bitsAsInt
}

func NewAppender() xorAppender {
	return xorAppender{b: bstream{stream: make([]byte, 2, 2)}}
}
