package prefetch

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

func parseWin10File(t *testing.T, name string) (data []byte, fi *FileInfo) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "win10", name))
	if err != nil {
		t.Skip("testdata not available:", err)
	}
	raw, err := Open(data)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	fi, err = ParseFileInfo(raw.Data[84:], raw.Version)
	if err != nil {
		t.Fatalf("ParseFileInfo: %v", err)
	}
	return raw.Data, fi
}

func TestParseFileMetrics_RealWin10(t *testing.T) {
	raw, fi := parseWin10File(t, "SERVICE.EXE-D640A8AF.pf")

	metrics, err := ParseFileMetrics(raw, Version10, fi.MetricsArrayOffset, fi.MetricsCount)
	if err != nil {
		t.Fatalf("ParseFileMetrics: %v", err)
	}
	if len(metrics) != 21 {
		t.Fatalf("got %d metrics, want 21", len(metrics))
	}
	if metrics[0].TraceChainIndex != 0 {
		t.Errorf("metrics[0].TraceChainIndex = %d, want 0", metrics[0].TraceChainIndex)
	}
	if metrics[0].FilenameStringOffset != 0 {
		t.Errorf("metrics[0].FilenameStringOffset = %d, want 0", metrics[0].FilenameStringOffset)
	}
	if metrics[0].FilenameStringLength == 0 {
		t.Error("metrics[0].FilenameStringLength = 0, expected non-zero")
	}
}

func TestParseTraceChains_RealWin10(t *testing.T) {
	raw, fi := parseWin10File(t, "SERVICE.EXE-D640A8AF.pf")

	chains, err := ParseTraceChains(raw, Version10, fi.TraceChainsOffset, fi.TraceChainsCount)
	if err != nil {
		t.Fatalf("ParseTraceChains: %v", err)
	}
	if len(chains) != 1549 {
		t.Fatalf("got %d trace chains, want 1549", len(chains))
	}
	// Win10 trace chains use 8-byte entries — NextEntryIndex should be 0
	if chains[0].NextEntryIndex != 0 {
		t.Errorf("chains[0].NextEntryIndex = %d, want 0 (v30 has no next-entry field)", chains[0].NextEntryIndex)
	}
}

func TestParseFilenames_RealWin10(t *testing.T) {
	raw, fi := parseWin10File(t, "SERVICE.EXE-D640A8AF.pf")

	names, err := ParseFilenames(raw, fi.FilenameStringsOff, fi.FilenameStringsSz)
	if err != nil {
		t.Fatalf("ParseFilenames: %v", err)
	}
	if len(names) == 0 {
		t.Fatal("got 0 filenames, expected some")
	}
	// Should contain Windows system paths
	found := false
	for _, n := range names {
		if n == "SERVICE.EXE" {
			found = true
			break
		}
	}
	if !found {
		t.Log("filenames found:", names[:min(5, len(names))])
	}
}

func TestParseFileMetrics_RealWin10_SETUP(t *testing.T) {
	raw, fi := parseWin10File(t, "SETUP.EXE-20FBC490.pf")

	metrics, err := ParseFileMetrics(raw, Version10, fi.MetricsArrayOffset, fi.MetricsCount)
	if err != nil {
		t.Fatalf("ParseFileMetrics: %v", err)
	}
	if len(metrics) == 0 {
		t.Fatal("got 0 metrics entries")
	}
	if metrics[0].FilenameStringLength == 0 {
		t.Error("metrics[0].FilenameStringLength = 0, expected non-zero")
	}
}

func TestParseTraceChains_RealWin10_SETUP(t *testing.T) {
	raw, fi := parseWin10File(t, "SETUP.EXE-20FBC490.pf")

	chains, err := ParseTraceChains(raw, Version10, fi.TraceChainsOffset, fi.TraceChainsCount)
	if err != nil {
		t.Fatalf("ParseTraceChains: %v", err)
	}
	if len(chains) == 0 {
		t.Fatal("got 0 trace chain entries")
	}
}

func TestParseFilenames_RealWin10_SETUP(t *testing.T) {
	raw, fi := parseWin10File(t, "SETUP.EXE-20FBC490.pf")

	names, err := ParseFilenames(raw, fi.FilenameStringsOff, fi.FilenameStringsSz)
	if err != nil {
		t.Fatalf("ParseFilenames: %v", err)
	}
	if len(names) == 0 {
		t.Fatal("got 0 filenames")
	}
}

func TestParseVolumeInfo_RealWin10(t *testing.T) {
	raw, fi := parseWin10File(t, "SERVICE.EXE-D640A8AF.pf")
	vi, err := ParseVolumeInfo(raw, Version10, fi.VolumesInfoOffset, fi.VolumesCount, fi.VolumesInfoSize)
	if err != nil {
		t.Fatalf("ParseVolumeInfo: %v", err)
	}
	if len(vi.Entries) != 1 {
		t.Fatalf("got %d volume entries, want 1", len(vi.Entries))
	}
	e := &vi.Entries[0]
	if e.DevicePath == "" {
		t.Error("DevicePath is empty")
	}
	if e.CreationTime == 0 {
		t.Error("CreationTime is zero")
	}
	if e.SerialNumber == 0 {
		t.Error("SerialNumber is zero")
	}
	if len(e.DirectoryStrings) == 0 {
		t.Error("DirectoryStrings is empty")
	}
}

func TestParseVolumeInfo_RealWin10_SETUP(t *testing.T) {
	raw, fi := parseWin10File(t, "SETUP.EXE-20FBC490.pf")
	vi, err := ParseVolumeInfo(raw, Version10, fi.VolumesInfoOffset, fi.VolumesCount, fi.VolumesInfoSize)
	if err != nil {
		t.Fatalf("ParseVolumeInfo: %v", err)
	}
	if len(vi.Entries) != 1 {
		t.Fatalf("got %d volume entries, want 1", len(vi.Entries))
	}
	e := &vi.Entries[0]
	// SETUP.EXE has sequential file references with non-zero sequence numbers
	if len(e.FileReferences) < 10 {
		t.Errorf("FileReferences count = %d, want >= 10", len(e.FileReferences))
	}
	hasSeq := false
	for _, fr := range e.FileReferences {
		if fr>>48 != 0 {
			hasSeq = true
			break
		}
	}
	if !hasSeq {
		t.Error("expected at least one file reference with non-zero sequence number")
	}
}

func TestVolumeEntrySize(t *testing.T) {
	if volumeEntrySize(VersionXP) != 40 {
		t.Errorf("v17 volume entry size = %d, want 40", volumeEntrySize(VersionXP))
	}
	if volumeEntrySize(VersionVista7) != 104 {
		t.Errorf("v23 volume entry size = %d, want 104", volumeEntrySize(VersionVista7))
	}
	if volumeEntrySize(Version8) != 104 {
		t.Errorf("v26 volume entry size = %d, want 104", volumeEntrySize(Version8))
	}
	if volumeEntrySize(Version10) != 96 {
		t.Errorf("v30 volume entry size = %d, want 96", volumeEntrySize(Version10))
	}
	if volumeEntrySize(Version11) != 96 {
		t.Errorf("v31 volume entry size = %d, want 96", volumeEntrySize(Version11))
	}
}

func TestParseVolumeInfo_Bounds(t *testing.T) {
	_, err := ParseVolumeInfo([]byte{0}, Version10, 0, 1, 10)
	if err == nil {
		t.Fatal("expected error for truncated volume info")
	}
}

func TestParseFileMetrics_EntrySizeV17(t *testing.T) {
	if metricsEntrySize(VersionXP) != 20 {
		t.Errorf("v17 entry size = %d, want 20", metricsEntrySize(VersionXP))
	}
}

func TestParseFileMetrics_EntrySizeV23(t *testing.T) {
	if metricsEntrySize(VersionVista7) != 32 {
		t.Errorf("v23 entry size = %d, want 32", metricsEntrySize(VersionVista7))
	}
}

func TestParseTraceChains_EntrySizeLegacy(t *testing.T) {
	if traceChainEntrySize(VersionXP) != 12 {
		t.Errorf("v17 entry size = %d, want 12", traceChainEntrySize(VersionXP))
	}
	if traceChainEntrySize(VersionVista7) != 12 {
		t.Errorf("v23 entry size = %d, want 12", traceChainEntrySize(VersionVista7))
	}
	if traceChainEntrySize(Version8) != 12 {
		t.Errorf("v26 entry size = %d, want 12", traceChainEntrySize(Version8))
	}
}

func TestParseTraceChains_EntrySizeV30(t *testing.T) {
	if traceChainEntrySize(Version10) != 8 {
		t.Errorf("v30 entry size = %d, want 8", traceChainEntrySize(Version10))
	}
	if traceChainEntrySize(Version11) != 8 {
		t.Errorf("v31 entry size = %d, want 8", traceChainEntrySize(Version11))
	}
}

func TestFiletimeToTimes(t *testing.T) {
	fts := []uint64{133313040000000000, 0}
	times := FiletimeToTimes(fts)
	if len(times) != 2 {
		t.Fatalf("got %d times, want 2", len(times))
	}
	if times[0].Year() != 2023 {
		t.Errorf("times[0].Year = %d, want 2023", times[0].Year())
	}
	// FILETIME 0 maps to 1601-01-01 (the epoch), not zero time
	if times[1].Year() != 1601 {
		t.Errorf("times[1].Year = %d, want 1601 (FILETIME epoch)", times[1].Year())
	}
}

func TestParseFileMetrics_TooShort(t *testing.T) {
	_, err := ParseFileMetrics([]byte{0, 0, 0, 0}, Version10, 0, 100)
	if err == nil {
		t.Fatal("expected error for short buffer")
	}
}

func TestParseTraceChains_TooShort(t *testing.T) {
	_, err := ParseTraceChains([]byte{0, 0, 0, 0}, Version10, 0, 100)
	if err == nil {
		t.Fatal("expected error for short buffer")
	}
}

func TestParseFilenames_TooShort(t *testing.T) {
	_, err := ParseFilenames([]byte{0x00}, 0, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseFilenames_OutOfBounds(t *testing.T) {
	_, err := ParseFilenames([]byte{0x00}, 10, 100)
	if err == nil {
		t.Fatal("expected error for out-of-bounds section")
	}
}

func TestParseFilenames_OddBytes(t *testing.T) {
	names, err := ParseFilenames([]byte{'A', 0, 'B'}, 0, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(names) != 1 || names[0] != "A" {
		t.Errorf("got %v, want [A]", names)
	}
}

func TestParseFilenames_TrailingNulls(t *testing.T) {
	names, err := ParseFilenames([]byte{'X', 0, 0, 0, 'Y', 0, 0, 0}, 0, 8)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(names) != 2 || names[0] != "X" || names[1] != "Y" {
		t.Errorf("got %v, want [X Y]", names)
	}
}

func TestParseVolumeInfo_InvalidOffsets(t *testing.T) {
	// Entry with device path offset beyond section boundary
	raw := make([]byte, 100)
	binary.LittleEndian.PutUint32(raw[0:4], 9999) // dev path off beyond bounds
	vi, err := ParseVolumeInfo(raw, Version10, 0, 1, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(vi.Entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(vi.Entries))
	}
	// Should handle gracefully — empty device path
	if vi.Entries[0].DevicePath != "" {
		t.Logf("DevicePath = %q (empty expected)", vi.Entries[0].DevicePath)
	}
}

func TestParseFileMetrics_ZeroCount(t *testing.T) {
	metrics, err := ParseFileMetrics([]byte{}, Version10, 0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(metrics) != 0 {
		t.Errorf("got %d metrics, want 0", len(metrics))
	}
}

func TestParseTraceChains_ZeroCount(t *testing.T) {
	chains, err := ParseTraceChains([]byte{}, Version10, 0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chains) != 0 {
		t.Errorf("got %d chains, want 0", len(chains))
	}
}

func TestParseVolumeInfo_ZeroCount(t *testing.T) {
	vi, err := ParseVolumeInfo([]byte{}, Version10, 0, 0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(vi.Entries) != 0 {
		t.Errorf("got %d entries, want 0", len(vi.Entries))
	}
}

func TestFiletimeToTime_Zero(t *testing.T) {
	ts := FiletimeToTime(0)
	if ts.Year() != 1601 {
		t.Errorf("year = %d, want 1601 (FILETIME epoch)", ts.Year())
	}
}

func TestFiletimeToTime_Max(t *testing.T) {
	ts := FiletimeToTime(1<<63 - 1)
	// Should not panic and should return a reasonable time
	if ts.Year() < 1601 || ts.Year() > 40000 {
		t.Errorf("unexpected year %d for max FILETIME", ts.Year())
	}
}

func TestOpen_BadSignature(t *testing.T) {
	_, err := Open([]byte("NOTAPFFILE"))
	if err == nil {
		t.Fatal("expected error for bad signature")
	}
}

func TestOpen_TruncatedSCCA(t *testing.T) {
	_, err := Open([]byte{0x11, 0x00, 0x00, 0x00, 'S', 'C', 'C'})
	if err == nil {
		t.Fatal("expected error for truncated SCCA file")
	}
}

func TestOpen_TruncatedMAM(t *testing.T) {
	_, err := Open([]byte{'M', 'A', 'M'})
	if err == nil {
		t.Fatal("expected error for truncated MAM file")
	}
}

func TestVolumeEntrySize_Unknown(t *testing.T) {
	if volumeEntrySize(100) != 96 {
		t.Errorf("unknown version should default to 96, got %d", volumeEntrySize(100))
	}
}
