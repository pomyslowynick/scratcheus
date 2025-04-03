package storage

import (
	"encoding/binary"
	"io"
	"math"
	"math/bits"
)

type XorAppender struct {
	b *Bstream

	t      int64
	v      float64
	tDelta uint64

	leading  uint8
	trailing uint8
}

func NewXorAppender() XorAppender {
	b := make([]byte, 2, 128)
	return XorAppender{
		b:        &Bstream{stream: b, count: 0},
		t:        1720000,
		v:        12.7,
		tDelta:   12321321,
		leading:  120,
		trailing: 0,
	}
}
func (a *XorAppender) Append(t int64, v float64) {
	var tDelta uint64
	num := binary.BigEndian.Uint16(a.b.bytes())
	switch num {
	case 0:
		buf := make([]byte, binary.MaxVarintLen64)
		for _, b := range buf[:binary.PutVarint(buf, t)] {
			a.b.writeByte(b)
		}
		a.b.writeBits(math.Float64bits(v), 64)
	case 1:
		tDelta = uint64(t - a.t)

		buf := make([]byte, binary.MaxVarintLen64)
		for _, b := range buf[:binary.PutUvarint(buf, tDelta)] {
			a.b.writeByte(b)
		}

		a.writeVDelta(v)
	default:
		tDelta = uint64(t - a.t)
		dod := int64(tDelta - a.tDelta)

		switch {
		case dod == 0:
			a.b.writeBit(zero)
		case bitRange(dod, 14):
			a.b.writeByte(0b10<<6 | (uint8(dod>>8) & (1<<6 - 1))) // 0b10 size code combined with 6 bits of dod.
			a.b.writeByte(uint8(dod))                             // Bottom 8 bits of dod.
		case bitRange(dod, 17):
			a.b.writeBits(0b110, 3)
			a.b.writeBits(uint64(dod), 17)
		case bitRange(dod, 20):
			a.b.writeBits(0b1110, 4)
			a.b.writeBits(uint64(dod), 20)
		default:
			a.b.writeBits(0b1111, 4)
			a.b.writeBits(uint64(dod), 64)
		}

		a.writeVDelta(v)
	}

	a.t = t
	a.v = v
	binary.BigEndian.PutUint16(a.b.bytes(), num+1)
	a.tDelta = tDelta
}

// bitRange returns whether the given integer can be represented by nbits.
// See docs/bstream.md.
func bitRange(x int64, nbits uint8) bool {
	return -((1<<(nbits-1))-1) <= x && x <= 1<<(nbits-1)
}

func (a *XorAppender) writeVDelta(v float64) {
	xorWrite(a.b, v, a.v, &a.leading, &a.trailing)
}

func xorWrite(b *Bstream, newValue, currentValue float64, leading, trailing *uint8) {
	delta := math.Float64bits(newValue) ^ math.Float64bits(currentValue)

	if delta == 0 {
		b.writeBit(zero)
		return
	}
	b.writeBit(one)

	newLeading := uint8(bits.LeadingZeros64(delta))
	newTrailing := uint8(bits.TrailingZeros64(delta))

	// Clamp number of leading zeros to avoid overflow when encoding.
	if newLeading >= 32 {
		newLeading = 31
	}

	if *leading != 0xff && newLeading >= *leading && newTrailing >= *trailing {
		// In this case, we stick with the current leading/trailing.
		b.writeBit(zero)
		b.writeBits(delta>>*trailing, 64-int(*leading)-int(*trailing))
		return
	}

	// Update leading/trailing for the caller.
	*leading, *trailing = newLeading, newTrailing

	b.writeBit(one)
	b.writeBits(uint64(newLeading), 5)

	// Note that if newLeading == newTrailing == 0, then sigbits == 64. But
	// that value doesn't actually fit into the 6 bits we have.  Luckily, we
	// never need to encode 0 significant bits, since that would put us in
	// the other case (vdelta == 0).  So instead we write out a 0 and adjust
	// it back to 64 on unpacking.
	sigbits := 64 - newLeading - newTrailing
	b.writeBits(uint64(sigbits), 6)
	b.writeBits(delta>>newTrailing, int(sigbits))
}

// Bstream is a stream of bits.
type Bstream struct {
	stream []byte // The data stream.
	count  uint8  // How many right-most bits are available for writing in the current byte (the last byte of the stream).
}

// Reset resets b around stream.
func (b *Bstream) Reset(stream []byte) {
	b.stream = stream
	b.count = 0
}

func (b *Bstream) bytes() []byte {
	return b.stream
}

type bit bool

const (
	zero bit = false
	one  bit = true
)

func (b *Bstream) writeBit(bit bit) {
	if b.count == 0 {
		b.stream = append(b.stream, 0)
		b.count = 8
	}

	i := len(b.stream) - 1

	if bit {
		b.stream[i] |= 1 << (b.count - 1)
	}

	b.count--
}

func (b *Bstream) writeByte(byt byte) {
	if b.count == 0 {
		b.stream = append(b.stream, byt)
		return
	}

	i := len(b.stream) - 1

	// Complete the last byte with the leftmost b.count bits from byt.
	b.stream[i] |= byt >> (8 - b.count)

	// Write the remainder, if any.
	b.stream = append(b.stream, byt<<b.count)
}

// writeBits writes the nbits right-most bits of u to the stream
// in left-to-right order.
func (b *Bstream) writeBits(u uint64, nbits int) {
	u <<= 64 - uint(nbits)
	for nbits >= 8 {
		byt := byte(u >> 56)
		b.writeByte(byt)
		u <<= 8
		nbits -= 8
	}

	for nbits > 0 {
		b.writeBit((u >> 63) == 1)
		u <<= 1
		nbits--
	}
}

type BstreamReader struct {
	stream       []byte
	streamOffset int // The offset from which read the next byte from the stream.

	buffer uint64 // The current buffer, filled from the stream, containing up to 8 bytes from which read bits.
	valid  uint8  // The number of right-most bits valid to read (from left) in the current 8 byte buffer.
	last   byte   // A copy of the last byte of the stream.
}

func newBReader(b []byte) BstreamReader {
	// The last byte of the stream can be updated later, so we take a copy.
	var last byte
	if len(b) > 0 {
		last = b[len(b)-1]
	}
	return BstreamReader{
		stream: b,
		last:   last,
	}
}

func (b *BstreamReader) readBit() (bit, error) {
	if b.valid == 0 {
		if !b.loadNextBuffer(1) {
			return false, io.EOF
		}
	}

	return b.readBitFast()
}

// readBitFast is like readBit but can return io.EOF if the internal buffer is empty.
// If it returns io.EOF, the caller should retry reading bits calling readBit().
// This function must be kept small and a leaf in order to help the compiler inlining it
// and further improve performances.
func (b *BstreamReader) readBitFast() (bit, error) {
	if b.valid == 0 {
		return false, io.EOF
	}

	b.valid--
	bitmask := uint64(1) << b.valid
	return (b.buffer & bitmask) != 0, nil
}

// readBits constructs a uint64 with the nbits right-most bits
// read from the stream, and any other bits 0.
func (b *BstreamReader) readBits(nbits uint8) (uint64, error) {
	if b.valid == 0 {
		if !b.loadNextBuffer(nbits) {
			return 0, io.EOF
		}
	}

	if nbits <= b.valid {
		return b.readBitsFast(nbits)
	}

	// We have to read all remaining valid bits from the current buffer and a part from the next one.
	bitmask := (uint64(1) << b.valid) - 1
	nbits -= b.valid
	v := (b.buffer & bitmask) << nbits
	b.valid = 0

	if !b.loadNextBuffer(nbits) {
		return 0, io.EOF
	}

	bitmask = (uint64(1) << nbits) - 1
	v |= ((b.buffer >> (b.valid - nbits)) & bitmask)
	b.valid -= nbits

	return v, nil
}

// readBitsFast is like readBits but can return io.EOF if the internal buffer is empty.
// If it returns io.EOF, the caller should retry reading bits calling readBits().
// This function must be kept small and a leaf in order to help the compiler inlining it
// and further improve performances.
func (b *BstreamReader) readBitsFast(nbits uint8) (uint64, error) {
	if nbits > b.valid {
		return 0, io.EOF
	}

	bitmask := (uint64(1) << nbits) - 1
	b.valid -= nbits

	return (b.buffer >> b.valid) & bitmask, nil
}

func (b *BstreamReader) ReadByte() (byte, error) {
	v, err := b.readBits(8)
	if err != nil {
		return 0, err
	}
	return byte(v), nil
}

// loadNextBuffer loads the next bytes from the stream into the internal buffer.
// The input nbits is the minimum number of bits that must be read, but the implementation
// can read more (if possible) to improve performances.
func (b *BstreamReader) loadNextBuffer(nbits uint8) bool {
	if b.streamOffset >= len(b.stream) {
		return false
	}

	// Handle the case there are more then 8 bytes in the buffer (most common case)
	// in a optimized way. It's guaranteed that this branch will never read from the
	// very last byte of the stream (which suffers race conditions due to concurrent
	// writes).
	if b.streamOffset+8 < len(b.stream) {
		b.buffer = binary.BigEndian.Uint64(b.stream[b.streamOffset:])
		b.streamOffset += 8
		b.valid = 64
		return true
	}

	// We're here if there are 8 or less bytes left in the stream.
	// The following code is slower but called less frequently.
	nbytes := int((nbits / 8) + 1)
	if b.streamOffset+nbytes > len(b.stream) {
		nbytes = len(b.stream) - b.streamOffset
	}

	buffer := uint64(0)
	skip := 0
	if b.streamOffset+nbytes == len(b.stream) {
		// There can be concurrent writes happening on the very last byte
		// of the stream, so use the copy we took at initialization time.
		buffer |= uint64(b.last)
		// Read up to the byte before
		skip = 1
	}

	for i := 0; i < nbytes-skip; i++ {
		buffer |= (uint64(b.stream[b.streamOffset+i]) << uint(8*(nbytes-i-1)))
	}

	b.buffer = buffer
	b.streamOffset += nbytes
	b.valid = uint8(nbytes * 8)

	return true
}
