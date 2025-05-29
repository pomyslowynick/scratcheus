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
			x.b.writeBits(uint64(dod), 7)
		case bitsRange(dod, 9):
			x.b.writeBits(0b110, 3)
			x.b.writeBits(uint64(dod), 9)
		case bitsRange(dod, 12):
			x.b.writeBits(0b1110, 4)
			x.b.writeBits(uint64(dod), 12)
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

type Sample struct {
	value     float64
	timestamp uint64
}

type Series struct {
	samples []Sample
}

func NewXorReader(s bstream) xorReader {
	return xorReader{stream: s}
}

// Refactor this function, turn it into smaller functions
func (x *xorReader) readSeries() Series {
	var samples []Sample

	si := NewIterator(x.stream)

	samplesNum := int((int16(si.nextByte()) << 8) + int16(si.nextByte()))

	//  I don't really like this switch statement, but it seems clearer than if statements + returns
	switch {
	case samplesNum == 0:
		// theoretically impossible, any series should should have this value set
		return Series{}
	case samplesNum == 1:
		samples = append(samples, x.readFirstSample(&si))
		return Series{samples: samples}
	case samplesNum == 2:
		samples = append(samples, x.readSecondSample(&si), x.readFirstSample(&si))
		return Series{samples: samples}
	case samplesNum > 2:
		samples = append(samples, x.readSecondSample(&si), x.readFirstSample(&si))
	}

	for i := 2; i != samplesNum; i++ {
		samples = append(samples, x.readSamples(&si))
	}

	return Series{samples: samples}
}

func (x *xorReader) readFirstSample(si *Iterator) Sample {

	// Read timestamp - no encoding
	timestampBytes := si.nextBytes(8)

	for i, byt := range timestampBytes {
		x.timestamp += uint64(byt) << ((7 - i) * 8)
	}

	// Read first float value - no encoding
	valueBytes := si.nextBytes(8)

	for i, byt := range valueBytes {
		x.tempValue += uint64(byt) << ((7 - i) * 8)
	}
	x.leadingZeros = bits.LeadingZeros64(x.tempValue)
	x.trailingZeros = bits.TrailingZeros64(x.tempValue)
	value := math.Float64frombits(x.tempValue)

	x.value = value

	return Sample{
		timestamp: x.timestamp,
		value:     x.value,
	}
}

func (x *xorReader) readSecondSample(si *Iterator) Sample {

	// Second tuple is always the first timestamp delta and xored value
	timestampDelta := si.nextBytes(8)
	for i, byt := range timestampDelta {
		x.ts_delta += uint64(byt) << ((7 - i) * 8)
	}
	x.timestamp = x.timestamp + x.ts_delta

	x.value = x.readXorEncodedValue(si)

	return Sample{
		timestamp: x.timestamp,
		value:     x.value,
	}
}

func (x *xorReader) readSamples(si *Iterator) Sample {
	//   Timestamps after first delta are deltas of deltas with variable encoding length
	var tsDod uint64
	tsDodFirstBit := si.nextBit()
	switch tsDodFirstBit {
	case zero:
		tsDod = 0
	case one:
		tsDodSecondBit := si.nextBit()
		switch tsDodSecondBit {
		case zero:
			tempDod := si.nextBits(7)
			tsDod = bitsSliceToInt(tempDod)
		case one:
			tsDodThirdBit := si.nextBit()
			switch tsDodThirdBit {
			case zero:
				tempDod := si.nextBits(9)
				tsDod = bitsSliceToInt(tempDod)
			case one:
				tsDodFourthBit := si.nextBit()
				switch tsDodFourthBit {
				case zero:
					tempDod := si.nextBits(12)
					tsDod = bitsSliceToInt(tempDod)
				case one:
					tempDod := si.nextBits(32)
					tsDod = bitsSliceToInt(tempDod)

				}
			}
		}

	}

	x.timestamp = x.timestamp + x.ts_delta + tsDod
	x.ts_delta = x.ts_delta + tsDod

	// Read rest of the values
	newValue := x.readXorEncodedValue(si)
	x.value = newValue

	return Sample{
		timestamp: x.timestamp,
		value:     x.value,
	}
}

func (x *xorReader) readXorEncodedValue(si *Iterator) float64 {
	var finalValue float64
	isDeltaZero := !bool(si.nextBit())

	// TODO: Double switch is a smell, there might be a better idiom for that
	switch {
	case isDeltaZero:
		return x.value
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
			sigBits := 64 - x.leadingZeros - x.trailingZeros
			xoredValue := si.nextBits(sigBits)

			xoredShiftedValue := bitsSliceToInt(xoredValue)
			xoredAsFloat := math.Float64frombits(xoredShiftedValue)

			return xoredAsFloat
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
