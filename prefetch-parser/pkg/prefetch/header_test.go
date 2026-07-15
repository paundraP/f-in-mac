package prefetch

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
	"unicode/utf16"
)

func uint32LE(v uint32) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, v)
	return b
}

func uint64LE(v uint64) []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, v)
	return b
}

func TestParseFileHeader_OK(t *testing.T) {
	raw := make([]byte, 84)
	binary.LittleEndian.PutUint32(raw[0:4], 17)          // version
	copy(raw[4:8], "SCCA")                                // signature
	binary.LittleEndian.PutUint32(raw[12:16], 10000)      // file size
	copy(raw[16:76], encodeUTF16("CALC.EXE"))             // exec name
	binary.LittleEndian.PutUint32(raw[76:80], 0xABCD1234) // hash

	h, err := ParseFileHeader(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h.Version != 17 {
		t.Errorf("Version = %d, want 17", h.Version)
	}
	if h.FileSize != 10000 {
		t.Errorf("FileSize = %d, want 10000", h.FileSize)
	}
	if h.ExecutableName != "CALC.EXE" {
		t.Errorf("ExecutableName = %q, want %q", h.ExecutableName, "CALC.EXE")
	}
	if h.PrefetchHash != 0xABCD1234 {
		t.Errorf("PrefetchHash = 0x%X, want 0xABCD1234", h.PrefetchHash)
	}
	if h.BootPrefetch {
		t.Error("BootPrefetch = true, want false")
	}
}

func TestParseFileHeader_BootPrefetch(t *testing.T) {
	raw := make([]byte, 84)
	binary.LittleEndian.PutUint32(raw[0:4], 17)
	copy(raw[4:8], "SCCA")
	binary.LittleEndian.PutUint32(raw[12:16], 10000)
	copy(raw[16:76], encodeUTF16("NTOSBOOT"))
	raw[80] = 0x01

	h, err := ParseFileHeader(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !h.BootPrefetch {
		t.Error("BootPrefetch = false, want true")
	}
}

func TestParseFileHeader_TooShort(t *testing.T) {
	_, err := ParseFileHeader(make([]byte, 10))
	if err == nil {
		t.Fatal("expected error for short header")
	}
}

func TestParseFileInfo_V17(t *testing.T) {
	raw := make([]byte, 68)
	copy(raw[0:4], uint32LE(152))   // metrics array offset
	copy(raw[4:8], uint32LE(5))     // metrics count
	copy(raw[8:12], uint32LE(300))  // trace chains offset
	copy(raw[12:16], uint32LE(5))   // trace chains count
	copy(raw[16:20], uint32LE(400)) // filename strings offset
	copy(raw[20:24], uint32LE(200)) // filename strings size
	copy(raw[24:28], uint32LE(500)) // volumes info offset
	copy(raw[28:32], uint32LE(1))   // volumes count
	copy(raw[32:36], uint32LE(256)) // volumes info size
	copy(raw[36:44], uint64LE(131000000000000000)) // last run time
	copy(raw[60:64], uint32LE(10))  // run count

	fi, err := ParseFileInfo(raw, VersionXP)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if fi.MetricsArrayOffset != 152 {
		t.Errorf("MetricsArrayOffset = %d, want 152", fi.MetricsArrayOffset)
	}
	if fi.MetricsCount != 5 {
		t.Errorf("MetricsCount = %d, want 5", fi.MetricsCount)
	}
	if fi.RunCount != 10 {
		t.Errorf("RunCount = %d, want 10", fi.RunCount)
	}
	if len(fi.LastRunTimes) != 1 {
		t.Fatalf("len(LastRunTimes) = %d, want 1", len(fi.LastRunTimes))
	}
	if fi.LastRunTimes[0] != 131000000000000000 {
		t.Errorf("LastRunTimes[0] = %d, want 131000000000000000", fi.LastRunTimes[0])
	}
}

func TestParseFileInfo_V23(t *testing.T) {
	raw := make([]byte, 156)
	copy(raw[0:4], uint32LE(240))  // metrics array offset
	copy(raw[4:8], uint32LE(3))    // metrics count
	copy(raw[8:12], uint32LE(400)) // trace chains offset
	copy(raw[12:16], uint32LE(3))  // trace chains count
	copy(raw[16:20], uint32LE(500))
	copy(raw[20:24], uint32LE(300))
	copy(raw[24:28], uint32LE(600))
	copy(raw[28:32], uint32LE(1))
	copy(raw[32:36], uint32LE(200))
	copy(raw[44:52], uint64LE(142000000000000000)) // last run time (offset 44 for v23)
	copy(raw[68:72], uint32LE(7))                  // run count (offset 68)

	fi, err := ParseFileInfo(raw, VersionVista7)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fi.MetricsArrayOffset != 240 {
		t.Errorf("MetricsArrayOffset = %d, want 240", fi.MetricsArrayOffset)
	}
	if fi.MetricsCount != 3 {
		t.Errorf("MetricsCount = %d, want 3", fi.MetricsCount)
	}
	if fi.RunCount != 7 {
		t.Errorf("RunCount = %d, want 7", fi.RunCount)
	}
	if len(fi.LastRunTimes) != 1 {
		t.Fatalf("len(LastRunTimes) = %d, want 1", len(fi.LastRunTimes))
	}
}

func TestParseFileInfo_V26(t *testing.T) {
	raw := make([]byte, 220)
	copy(raw[0:4], uint32LE(304)) // metrics array offset
	copy(raw[4:8], uint32LE(12))  // metrics count
	// last run times at offset 44, 8 entries
	copy(raw[44:52], uint64LE(100)) // t0
	copy(raw[52:60], uint64LE(200)) // t1
	copy(raw[60:68], uint64LE(300)) // t2
	copy(raw[68:76], uint64LE(400)) // t3
	copy(raw[76:84], uint64LE(500)) // t4
	copy(raw[84:92], uint64LE(600)) // t5
	copy(raw[92:100], uint64LE(700)) // t6
	copy(raw[100:108], uint64LE(800)) // t7
	copy(raw[124:128], uint32LE(99))  // run count

	fi, err := ParseFileInfo(raw, Version8)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fi.MetricsArrayOffset != 304 {
		t.Errorf("MetricsArrayOffset = %d, want 304", fi.MetricsArrayOffset)
	}
	if fi.MetricsCount != 12 {
		t.Errorf("MetricsCount = %d, want 12", fi.MetricsCount)
	}
	if fi.RunCount != 99 {
		t.Errorf("RunCount = %d, want 99", fi.RunCount)
	}
	if len(fi.LastRunTimes) != 8 {
		t.Fatalf("len(LastRunTimes) = %d, want 8", len(fi.LastRunTimes))
	}
	for i, v := range fi.LastRunTimes {
		want := uint64((i + 1) * 100)
		if v != want {
			t.Errorf("LastRunTimes[%d] = %d, want %d", i, v, want)
		}
	}
}

func TestParseFileInfo_TooShort(t *testing.T) {
	_, err := ParseFileInfo(make([]byte, 10), VersionXP)
	if err == nil {
		t.Fatal("expected error for short file info")
	}
}

func TestParseFileInfo_UnsupportedVersion(t *testing.T) {
	_, err := ParseFileInfo(make([]byte, 68), 99)
	if err == nil {
		t.Fatal("expected error for unsupported version")
	}
}

func TestFiletimeToTime(t *testing.T) {
	// FILETIME for 2023-06-15T12:00:00Z
	// unix = 1686830400, delta = 11644473600 sec
	// ft = (1686830400 + 11644473600) * 10000000 = 133313040000000000
	ft := uint64(133313040000000000)
	ts := FiletimeToTime(ft)
	if ts.Year() != 2023 || ts.Month() != 6 || ts.Day() != 15 {
		t.Errorf("got %v, want 2023-06-15", ts)
	}
	if ts.Hour() != 12 || ts.Minute() != 0 {
		t.Errorf("got %v, want 12:00", ts)
	}
}

func TestFileInfoOffsetsFor_V30Variant2(t *testing.T) {
	raw := make([]byte, 4)
	binary.LittleEndian.PutUint32(raw, 296) // variant 2: metrics offset = 0x128
	off, err := fileInfoOffsetsFor(raw, Version10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if off.SectionSize != 212 {
		t.Errorf("SectionSize = %d, want 212", off.SectionSize)
	}
	if off.RunCount != 116 {
		t.Errorf("RunCount field offset = %d, want 116", off.RunCount)
	}
	if off.HashStringOffset != 128 {
		t.Errorf("HashStringOffset field offset = %d, want 128", off.HashStringOffset)
	}
}

func TestFileInfoOffsetsFor_V30Variant1(t *testing.T) {
	raw := make([]byte, 4)
	binary.LittleEndian.PutUint32(raw, 304) // variant 1: metrics offset = 0x130
	off, err := fileInfoOffsetsFor(raw, Version10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if off.SectionSize != 220 {
		t.Errorf("SectionSize = %d, want 220", off.SectionSize)
	}
	if off.RunCount != 124 {
		t.Errorf("RunCount field offset = %d, want 124", off.RunCount)
	}
	if off.HashStringOffset != 136 {
		t.Errorf("HashStringOffset field offset = %d, want 136", off.HashStringOffset)
	}
}

func TestFileInfoOffsetsFor_V30TooShort(t *testing.T) {
	off, err := fileInfoOffsetsFor(nil, Version10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if off.SectionSize != 212 {
		t.Errorf("SectionSize = %d, want 212 (variant 2 default)", off.SectionSize)
	}
}

func TestFileInfoOffsetsFor_V31(t *testing.T) {
	// v31 defaults to variant 2 layout
	off, err := fileInfoOffsetsFor(nil, Version11)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if off.SectionSize != 212 {
		t.Errorf("SectionSize = %d, want 212", off.SectionSize)
	}
}

func TestParseFileInfo_V30_RealFile(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "win10", "SERVICE.EXE-D640A8AF.pf"))
	if err != nil {
		t.Skip("testdata not available:", err)
	}
	raw, err := Open(data)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	fi, err := ParseFileInfo(raw.Data[84:], raw.Version)
	if err != nil {
		t.Fatalf("ParseFileInfo: %v", err)
	}

	// Validate against known values from the real file
	if fi.MetricsArrayOffset != 296 {
		t.Errorf("MetricsArrayOffset = %d, want 296 (variant 2)", fi.MetricsArrayOffset)
	}
	if fi.MetricsCount != 21 {
		t.Errorf("MetricsCount = %d, want 21", fi.MetricsCount)
	}
	if fi.TraceChainsOffset != 968 {
		t.Errorf("TraceChainsOffset = %d, want 968", fi.TraceChainsOffset)
	}
	if fi.TraceChainsCount != 1549 {
		t.Errorf("TraceChainsCount = %d, want 1549", fi.TraceChainsCount)
	}
	if fi.FilenameStringsOff != 13360 {
		t.Errorf("FilenameStringsOff = %d, want 13360", fi.FilenameStringsOff)
	}
	if fi.FilenameStringsSz != 2714 {
		t.Errorf("FilenameStringsSz = %d, want 2714", fi.FilenameStringsSz)
	}
	if fi.VolumesInfoOffset != 16200 {
		t.Errorf("VolumesInfoOffset = %d, want 16200", fi.VolumesInfoOffset)
	}
	if fi.VolumesCount != 1 {
		t.Errorf("VolumesCount = %d, want 1", fi.VolumesCount)
	}
	if fi.VolumesInfoSize != 1150 {
		t.Errorf("VolumesInfoSize = %d, want 1150", fi.VolumesInfoSize)
	}
	if fi.RunCount != 1 {
		t.Errorf("RunCount = %d, want 1", fi.RunCount)
	}
	if len(fi.LastRunTimes) != 8 {
		t.Errorf("len(LastRunTimes) = %d, want 8", len(fi.LastRunTimes))
	}
	if fi.LastRunTimes[0] == 0 {
		t.Error("LastRunTimes[0] = 0, expected a real timestamp")
	}
	if fi.HashStringOffset != 16074 {
		t.Errorf("HashStringOffset = %d, want 16074", fi.HashStringOffset)
	}
	if fi.HashStringSize != 120 {
		t.Errorf("HashStringSize = %d, want 120", fi.HashStringSize)
	}
}

func encodeUTF16(s string) []byte {
	runes := []rune(s)
	u16 := utf16.Encode(runes)
	b := make([]byte, 60)
	for i, r := range u16 {
		if i*2+1 >= 60 {
			break
		}
		binary.LittleEndian.PutUint16(b[i*2:], r)
	}
	return b
}
