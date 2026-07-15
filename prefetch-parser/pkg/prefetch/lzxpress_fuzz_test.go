package prefetch

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func FuzzLZXpressHuffmanDecompress(f *testing.F) {
	// Seed corpus with the real Win10 prefetch compressed payload
	data, err := os.ReadFile(filepath.Join("testdata", "win10", "SERVICE.EXE-D640A8AF.pf"))
	if err == nil && len(data) > 8 {
		realOutputSize := int(data[4]) | int(data[5])<<8 | int(data[6])<<16 | int(data[7])<<24
		f.Add(data[8:], realOutputSize)
	}

	// Seed with minimal valid-looking inputs
	f.Add([]byte{}, 0)
	f.Add(bytes.Repeat([]byte{0xFF}, 260), 100)

	f.Fuzz(func(t *testing.T, compressed []byte, outputSize int) {
		if outputSize < 0 || outputSize > 10*1024*1024 {
			return
		}
		result, err := lzxpressHuffmanDecompress(compressed, outputSize)
		if err != nil && result != nil {
			t.Errorf("non-nil result with error: %v, len=%d", err, len(result))
		}
		if result != nil && len(result) > outputSize {
			t.Errorf("result len %d > outputSize %d", len(result), outputSize)
		}
	})
}
