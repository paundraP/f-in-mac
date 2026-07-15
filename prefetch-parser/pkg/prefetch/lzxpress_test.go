package prefetch

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLZXpressDecompressRealWin10(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "win10", "SERVICE.EXE-D640A8AF.pf"))
	if err != nil {
		t.Skip("testdata not available:", err)
	}

	raw, err := Open(data)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if !raw.WasCompressed {
		t.Fatal("expected compressed file")
	}
	if raw.Version != Version10 {
		t.Fatalf("Version = %d, want %d", raw.Version, Version10)
	}
	if len(raw.Data) != 17350 {
		t.Fatalf("decompressed size = %d, want 17350", len(raw.Data))
	}

	// Verify SCCA signature in decompressed output
	if string(raw.Data[4:8]) != "SCCA" {
		t.Fatal("decompressed payload missing SCCA signature")
	}
}

func TestLZXpressDecompressChunkTooShort(t *testing.T) {
	_, err := lzxpressHuffmanDecompress([]byte{0x00}, 100)
	if err == nil {
		t.Fatal("expected error for short input")
	}
}

func TestLZXpressDecompressCorruptTree(t *testing.T) {
	// 256 bytes of all-zero table => no Huffman codes defined
	input := make([]byte, 260)
	_, err := lzxpressHuffmanDecompress(input, 100)
	if err == nil {
		t.Fatal("expected error for corrupt Huffman tree")
	}
}

func TestOpenRealWin10(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "win10", "SERVICE.EXE-D640A8AF.pf"))
	if err != nil {
		t.Skip("testdata not available:", err)
	}

	raw, err := Open(data)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if raw.Version != Version10 {
		t.Errorf("Version = %d, want %d", raw.Version, Version10)
	}
	if !raw.WasCompressed {
		t.Error("WasCompressed = false, want true")
	}

	// Parse decompressed header
	h, err := ParseFileHeader(raw.Data)
	if err != nil {
		t.Fatalf("ParseFileHeader: %v", err)
	}
	if h.ExecutableName != "SERVICE.EXE" {
		t.Errorf("ExecutableName = %q, want %q", h.ExecutableName, "SERVICE.EXE")
	}
	if h.PrefetchHash != 0xD640A8AF {
		t.Errorf("PrefetchHash = 0x%08X, want 0xD640A8AF", h.PrefetchHash)
	}
	if h.BootPrefetch {
		t.Error("BootPrefetch = true, want false")
	}

	// Parse file info section
	fi, err := ParseFileInfo(raw.Data[84:], raw.Version)
	if err != nil {
		t.Fatalf("ParseFileInfo: %v", err)
	}
	if fi.RunCount != 1 {
		t.Errorf("RunCount = %d, want 1", fi.RunCount)
	}
	if len(fi.LastRunTimes) != 8 {
		t.Errorf("len(LastRunTimes) = %d, want 8", len(fi.LastRunTimes))
	}
	// First timestamp should be set, rest should be zero
	if fi.LastRunTimes[0] == 0 {
		t.Error("LastRunTimes[0] is zero, expected a real timestamp")
	}
	for i := 1; i < len(fi.LastRunTimes); i++ {
		if fi.LastRunTimes[i] != 0 {
			t.Errorf("LastRunTimes[%d] = %d, want 0", i, fi.LastRunTimes[i])
		}
	}
}
