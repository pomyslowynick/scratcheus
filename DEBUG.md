Saving useful bits from debugging sessions:

## xor.go
Output of non zero value encoding with one bit diff, I feel like leading/trailing zeros should have been set beforehand with the first/second sample.

Keeping as a reference.
```
(dlv) p x.b.stream
[]uint8 len: 28, cap: 32, [0,2,0,0,0,0,104,14,30,162,64
,6,4,187,26,243,161,77,0,0,0,0,0,0,0,0,79,64]
(dlv) p %b x.b.stream
[]uint8 len: 28, cap: 32, [0,10,0,0,0,0,1101000,1110,11
110,10100010,1000000,110,100,10111011,11010,11110011,10
100001,1001101,0,0,0,0,0,0,0,0,1001111,1000000]
(dlv) n
> github.com/pomyslowynick/scratcheus/tsdb.(*xorAppende
r).writeVDelta() ./tsdb/xor.go:111 (PC: 0x5a0f76)
   106:                 x.b.writeBits(delta, 64-(leadin
g_zeros+trailing_zeros))
   107:                 return
   108:         }
   109:
   110:         x.b.writeBit(one)
=> 111:         x.b.writeBits(uint64(leading_zeros), 5)
   112:         x.b.writeBits(uint64(64-(leading_zeros+
trailing_zeros)), 6)
   113:         x.b.writeBits(delta>>trailing_zeros, 64
-(leading_zeros+trailing_zeros))
   114:
   115:         x.leading_zeros = leading_zeros
   116:         x.trailing_zeros = trailing_zeros
(dlv) p %b x.b.stream
[]uint8 len: 28, cap: 32, [0,10,0,0,0,0,1101000,1110,11
110,10100010,1000000,110,100,10111011,11010,11110011,10
100001,1001101,0,0,0,0,0,0,0,0,1001111,1100000]
(dlv) n
> github.com/pomyslowynick/scratcheus/tsdb.(*xorAppende
r).writeVDelta() ./tsdb/xor.go:112 (PC: 0x5a0f94)
   107:                 return
   108:         }
   109:
   110:         x.b.writeBit(one)
   111:         x.b.writeBits(uint64(leading_zeros), 5)
=> 112:         x.b.writeBits(uint64(64-(leading_zeros+
trailing_zeros)), 6)
   113:         x.b.writeBits(delta>>trailing_zeros, 64
-(leading_zeros+trailing_zeros))
   114:
   115:         x.leading_zeros = leading_zeros
   116:         x.trailing_zeros = trailing_zeros
   117: }
(dlv) p %b x.b.stream
[]uint8 len: 28, cap: 32, [0,10,0,0,0,0,1101000,1110,11
110,10100010,1000000,110,100,10111011,11010,11110011,10
100001,1001101,0,0,0,0,0,0,0,0,1001111,1101100]
(dlv) n
> github.com/pomyslowynick/scratcheus/tsdb.(*xorAppende
r).writeVDelta() ./tsdb/xor.go:113 (PC: 0x5a0fbe)
   108:         }
   109:
   110:         x.b.writeBit(one)
   111:         x.b.writeBits(uint64(leading_zeros), 5)
   112:         x.b.writeBits(uint64(64-(leading_zeros+
trailing_zeros)), 6)
=> 113:         x.b.writeBits(delta>>trailing_zeros, 64
-(leading_zeros+trailing_zeros))
   114:
   115:         x.leading_zeros = leading_zeros
   116:         x.trailing_zeros = trailing_zeros
   117: }
   118:
(dlv) p %b x.b.stream
[]uint8 len: 29, cap: 32, [0,10,0,0,0,0,1101000,1110,11
110,10100010,1000000,110,100,10111011,11010,11110011,10
100001,1001101,0,0,0,0,0,0,0,0,1001111,1101100,100]
(dlv) n
> github.com/pomyslowynick/scratcheus/tsdb.(*xorAppende
r).writeVDelta() ./tsdb/xor.go:115 (PC: 0x5a1005)
   110:         x.b.writeBit(one)
   111:         x.b.writeBits(uint64(leading_zeros), 5)
   112:         x.b.writeBits(uint64(64-(leading_zeros+
trailing_zeros)), 6)
   113:         x.b.writeBits(delta>>trailing_zeros, 64
-(leading_zeros+trailing_zeros))
   114:
=> 115:         x.leading_zeros = leading_zeros
   116:         x.trailing_zeros = trailing_zeros
   117: }
   118:
   119: func (x *xorAppender) Series() []byte {
   120:         return x.b.stream
(dlv) p %b x.b.stream
[]uint8 len: 29, cap: 32, [0,10,0,0,0,0,1101000,1110,11
110,10100010,1000000,110,100,10111011,11010,11110011,10
100001,1001101,0,0,0,0,0,0,0,0,1001111,1101100,110]
```
