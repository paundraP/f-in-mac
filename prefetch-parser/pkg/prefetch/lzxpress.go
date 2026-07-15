package prefetch

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"sort"
)

type bitReader struct {
	src  []byte
	off  int
	buf  uint32
	bits uint32
}

func newBitReader(src []byte, off int) (*bitReader, error) {
	if off+4 > len(src) {
		return nil, io.ErrUnexpectedEOF
	}
	return &bitReader{
		src:  src,
		off:  off + 4,
		buf:  uint32(binary.LittleEndian.Uint16(src[off:]))<<16 |
			uint32(binary.LittleEndian.Uint16(src[off+2:])),
		bits: 32,
	}, nil
}

func (br *bitReader) peek(n uint32) uint32 {
	if n == 0 {
		return 0
	}
	return br.buf >> (32 - n)
}

func (br *bitReader) skip(n uint32) error {
	br.buf <<= n
	br.bits -= n
	if br.bits < 16 {
		if br.off+2 > len(br.src) {
			return io.EOF
		}
		br.buf |= uint32(binary.LittleEndian.Uint16(br.src[br.off:])) << (16 - br.bits)
		br.off += 2
		br.bits += 16
	}
	return nil
}

type nodeIdx uint32

type huffNode struct {
	leaf   bool
	symbol uint32
	child  [2]nodeIdx
}

const rootIdx nodeIdx = 0

type huffTree struct {
	nodes []huffNode
	next  nodeIdx
}

func newHuffTree() *huffTree {
	t := &huffTree{nodes: make([]huffNode, 1024)}
	t.next = 1
	t.nodes[0] = huffNode{child: [2]nodeIdx{0, 0}}
	return t
}

func (t *huffTree) addLeaf(symbol uint32, code uint32, codeLen uint32) error {
	ni := rootIdx
	for i := codeLen - 1; i > 0; i-- {
		bit := (code >> i) & 1
		n := &t.nodes[ni]
		if n.child[bit] == 0 {
			if int(t.next) >= len(t.nodes) {
				return errors.New("prefetch: huffman tree overflow")
			}
			n.child[bit] = t.next
			t.nodes[t.next] = huffNode{child: [2]nodeIdx{0, 0}}
			t.next++
		}
		ni = n.child[bit]
	}
	lastBit := code & 1
	n := &t.nodes[ni]
	if n.child[lastBit] != 0 {
		return errors.New("prefetch: conflicting huffman code")
	}
	if int(t.next) >= len(t.nodes) {
		return errors.New("prefetch: huffman tree overflow")
	}
	n.child[lastBit] = t.next
	t.nodes[t.next] = huffNode{leaf: true, symbol: symbol}
	t.next++
	return nil
}

func (t *huffTree) decodeSymbol(br *bitReader) (uint32, error) {
	ni := rootIdx
	for {
		bit, err := br.read(1)
		if err != nil {
			return 0, err
		}
		next := t.nodes[ni].child[bit]
		if next == 0 {
			return 0, errors.New("prefetch: bad huffman code")
		}
		n := &t.nodes[next]
		if n.leaf {
			return n.symbol, nil
		}
		ni = next
	}
}

type codeSym struct {
	symbol uint32
	length uint32
}

func buildHuffTree(table []byte) (*huffTree, error) {
	var syms [512]codeSym
	for i := 0; i < 256; i++ {
		v := table[i]
		syms[2*i] = codeSym{symbol: uint32(2 * i), length: uint32(v & 0x0f)}
		syms[2*i+1] = codeSym{symbol: uint32(2*i + 1), length: uint32(v >> 4)}
	}

	sort.SliceStable(syms[:], func(i, j int) bool {
		a, b := &syms[i], &syms[j]
		if a.length != b.length {
			return a.length < b.length
		}
		return a.symbol < b.symbol
	})

	idx := 0
	for idx < 512 && syms[idx].length == 0 {
		idx++
	}

	tree := newHuffTree()
	mask := uint32(0)
	prevLen := uint32(1)

	for ; idx < 512; idx++ {
		s := &syms[idx]
		mask <<= s.length - prevLen
		prevLen = s.length
		if err := tree.addLeaf(s.symbol, mask, s.length); err != nil {
			return nil, err
		}
		mask++
	}

	return tree, nil
}

func (br *bitReader) read(n uint32) (uint32, error) {
	v := br.peek(n)
	return v, br.skip(n)
}

func decompressChunk(input []byte, inOff int, output []byte, outOff int, chunkSize int) (int, int, error) {
	if inOff+256 > len(input) {
		return 0, 0, io.ErrUnexpectedEOF
	}

	tree, err := buildHuffTree(input[inOff:])
	if err != nil {
		return 0, 0, fmt.Errorf("prefetch: build huffman tree: %w", err)
	}

	br, err := newBitReader(input, inOff+256)
	if err != nil {
		return 0, 0, err
	}

	i := outOff
	end := outOff + chunkSize
	if end > len(output) {
		end = len(output)
	}

	for i < end {
		sym, err := tree.decodeSymbol(br)
		if err != nil {
			if err == io.EOF {
				err = nil
			}
			return br.off, i, err
		}

		if sym < 256 {
			output[i] = byte(sym)
			i++
			continue
		}

		sym -= 256
		length := sym & 0x0f
		extraBits := sym >> 4

		if length == 15 {
			if br.off >= len(br.src) {
				return br.off, i, io.ErrUnexpectedEOF
			}
			length += uint32(br.src[br.off])
			br.off++
			if length == 270 {
				if br.off+2 > len(br.src) {
					return br.off, i, io.ErrUnexpectedEOF
				}
				length = uint32(binary.LittleEndian.Uint16(br.src[br.off:]))
				br.off += 2
			}
		}

		rawOff := int32(br.peek(extraBits))
		if err := br.skip(extraBits); err != nil {
			return br.off, i, err
		}
		rawOff |= 1 << extraBits
		rawOff = -rawOff

		length += 3
		for k := uint32(0); k < length; k++ {
			srcIdx := i + int(rawOff)
			if srcIdx < 0 || srcIdx >= len(output) || i >= len(output) {
				return br.off, i, fmt.Errorf("prefetch: decompress bounds: src=%d dst=%d cap=%d", srcIdx, i, len(output))
			}
			output[i] = output[srcIdx]
			i++
		}
	}

	return br.off, i, nil
}

func lzxpressHuffmanDecompress(input []byte, outputSize int) ([]byte, error) {
	output := make([]byte, outputSize)
	inOff := 0
	outOff := 0

	for outOff < outputSize && inOff < len(input) {
		chunkSize := outputSize - outOff
		if chunkSize > 65536 {
			chunkSize = 65536
		}

		var err error
		inOff, outOff, err = decompressChunk(input, inOff, output, outOff, chunkSize)
		if err != nil {
			return nil, fmt.Errorf("prefetch: decompress at inOff=%d: %w", inOff, err)
		}
	}

	return output, nil
}
