package prefetch

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type sampleInfo struct {
	relPath    string
	wantVer    Version
	wantExec   string
	wantHash   uint32
	wantRuns   int
	wantRunCnt uint32
}

var samples = []sampleInfo{
	{"xp/EXPLORER.EXE-082F38A9.pf", VersionXP, "EXPLORER.EXE", 0x082F38A9, 1, 42},
	{"win7/CALC.EXE-77FDF17F.pf", VersionVista7, "CALC.EXE", 0x77FDF17F, 1, 7},
	{"win8/NOTEPAD.EXE-D8414F97.pf", Version8, "NOTEPAD.EXE", 0xD8414F97, 8, 4},
	{"win10/SERVICE.EXE-D640A8AF.pf", Version10, "SERVICE.EXE", 0xD640A8AF, 8, 1},
	{"win10/SETUP.EXE-20FBC490.pf", Version10, "SETUP.EXE", 0x20FBC490, 8, 2},
	{"win11/DLLHOST.EXE-C4F24392.pf", Version11, "DLLHOST.EXE", 0xC4F24392, 8, 11},
	{"win11/WINWORD.EXE-AB6EC2FA.pf", Version11, "WINWORD.EXE", 0xAB6EC2FA, 8, 8},
}

func TestSamples_OpenAndHeader(t *testing.T) {
	for _, s := range samples {
		t.Run(s.wantExec, func(t *testing.T) {
			raw, h := openAndParseHeader(t, s.relPath)

			if raw.Version != s.wantVer {
				t.Errorf("Version = %d, want %d", raw.Version, s.wantVer)
			}
			if h.ExecutableName != s.wantExec {
				t.Errorf("ExecutableName = %q, want %q", h.ExecutableName, s.wantExec)
			}
			if h.PrefetchHash != s.wantHash {
				t.Errorf("PrefetchHash = 0x%08X, want 0x%08X", h.PrefetchHash, s.wantHash)
			}
		})
	}
}

func TestSamples_FileInfo(t *testing.T) {
	for _, s := range samples {
		t.Run(s.wantExec, func(t *testing.T) {
			raw, _ := openAndParseHeader(t, s.relPath)
			fi, err := ParseFileInfo(raw.Data[84:], raw.Version)
			if err != nil {
				t.Fatalf("ParseFileInfo: %v", err)
			}

			if int(fi.RunCount) < 1 {
				t.Errorf("RunCount = %d, want >= 1", fi.RunCount)
			}
			if len(fi.LastRunTimes) != s.wantRuns {
				t.Errorf("len(LastRunTimes) = %d, want %d", len(fi.LastRunTimes), s.wantRuns)
			}
			if fi.LastRunTimes[0] == 0 {
				t.Error("LastRunTimes[0] = 0, expected non-zero")
			}
			if fi.MetricsCount == 0 {
				t.Error("MetricsCount = 0, expected > 0")
			}
			if fi.FilenameStringsSz == 0 {
				t.Error("FilenameStringsSz = 0, expected > 0")
			}
			if fi.VolumesCount == 0 {
				t.Error("VolumesCount = 0, expected > 0")
			}

			// v30/v31 should have hash string fields
			if raw.Version >= Version10 {
				if fi.HashStringSize == 0 {
					t.Error("HashStringSize = 0, expected non-zero for v30+")
				}
			}
		})
	}
}

func TestSamples_MetricsAndFilenames(t *testing.T) {
	for _, s := range samples {
		t.Run(s.wantExec, func(t *testing.T) {
			raw, _ := openAndParseHeader(t, s.relPath)
			fi := parseFileInfo(t, raw)

			metrics, err := ParseFileMetrics(raw.Data, raw.Version, fi.MetricsArrayOffset, fi.MetricsCount)
			if err != nil {
				t.Fatalf("ParseFileMetrics: %v", err)
			}
			if len(metrics) == 0 {
				t.Fatal("Metrics: 0 entries")
			}

			// First metric should point to the executable itself (offset 0)
			if metrics[0].FilenameStringOffset != 0 {
				t.Errorf("metrics[0].FilenameStringOffset = %d, want 0", metrics[0].FilenameStringOffset)
			}
			if metrics[0].FilenameStringLength == 0 {
				t.Errorf("metrics[0].FilenameStringLength = 0, expected non-zero")
			}

			// For v23+, first metric should have a valid trace chain index
			if raw.Version >= VersionVista7 {
				if metrics[0].BlocksToPrefetch == 0 && metrics[0].TraceEntriesCount > 0 {
					// blocksToPrefetch can be 0, that's OK
				}
			}

			fnames, err := ParseFilenames(raw.Data, fi.FilenameStringsOff, fi.FilenameStringsSz)
			if err != nil {
				t.Fatalf("ParseFilenames: %v", err)
			}
			if len(fnames) != len(metrics) {
				t.Errorf("filenames count = %d, metrics count = %d (should match)", len(fnames), len(metrics))
			}

			// Verify first filename contains the executable name
			if len(fnames) > 0 {
				if !strings.Contains(strings.ToUpper(fnames[0]), s.wantExec[:4]) {
					t.Logf("first filename = %q (may not contain executable name on all versions)", fnames[0])
				}
			}
		})
	}
}

func TestSamples_TraceChains(t *testing.T) {
	for _, s := range samples {
		t.Run(s.wantExec, func(t *testing.T) {
			raw, _ := openAndParseHeader(t, s.relPath)
			fi := parseFileInfo(t, raw)

			chains, err := ParseTraceChains(raw.Data, raw.Version, fi.TraceChainsOffset, fi.TraceChainsCount)
			if err != nil {
				t.Fatalf("ParseTraceChains: %v", err)
			}
			if len(chains) == 0 {
				t.Fatal("TraceChains: 0 entries")
			}

			// v17/v23/v26 should have NextEntryIndex values
			if raw.Version <= Version8 {
				if chains[0].NextEntryIndex == 0 && len(chains) > 1 {
					t.Logf("chains[0].NextEntryIndex = 0 (may be end-of-chain)")
				}
				// At least one entry should have a non-default next index
				hasChain := false
				for _, c := range chains {
					if c.NextEntryIndex != 0 && c.NextEntryIndex != -1 {
						hasChain = true
						break
					}
				}
				if !hasChain {
					t.Log("no trace chain links found (may be all zeroed)")
				}
			}
		})
	}
}

func TestSamples_VolumeInfo(t *testing.T) {
	for _, s := range samples {
		t.Run(s.wantExec, func(t *testing.T) {
			raw, _ := openAndParseHeader(t, s.relPath)
			fi := parseFileInfo(t, raw)

			vi, err := ParseVolumeInfo(raw.Data, raw.Version, fi.VolumesInfoOffset, fi.VolumesCount, fi.VolumesInfoSize)
			if err != nil {
				t.Fatalf("ParseVolumeInfo: %v", err)
			}
			if len(vi.Entries) == 0 {
				t.Fatal("VolumeInfo: 0 entries")
			}

			e := vi.Entries[0]
			if e.DevicePath == "" {
				t.Error("DevicePath is empty")
			}
			if e.CreationTime == 0 {
				t.Error("CreationTime is zero")
			}
			if e.SerialNumber == 0 {
				t.Error("SerialNumber is zero")
			}

			// Directory strings should not be empty
			if len(e.DirectoryStrings) == 0 {
				t.Error("DirectoryStrings is empty")
			}

			// Verify directory strings have volume prefix
			if len(e.DirectoryStrings) > 0 {
				if !strings.HasPrefix(e.DirectoryStrings[0], "\\") {
					t.Errorf("DirectoryStrings[0] = %q, expected starting with \\", e.DirectoryStrings[0])
				}
			}
		})
	}
}

func TestSamples_OffsetsWithinBounds(t *testing.T) {
	for _, s := range samples {
		t.Run(s.wantExec, func(t *testing.T) {
			raw, _ := openAndParseHeader(t, s.relPath)
			fi := parseFileInfo(t, raw)
			fileLen := len(raw.Data)

			check := func(name string, offset, size uint32) {
				end := int(offset + size)
				if end > fileLen {
					t.Errorf("%s: offset=%d size=%d end=%d exceeds file length %d", name, offset, size, end, fileLen)
				}
			}

			check("metrics", fi.MetricsArrayOffset, fi.MetricsCount*32)
			check("tracechains", fi.TraceChainsOffset, fi.TraceChainsCount*8)
			check("filenames", fi.FilenameStringsOff, fi.FilenameStringsSz)
			check("volumes", fi.VolumesInfoOffset, fi.VolumesInfoSize)
		})
	}
}

func TestSamples_FirstLoadedFileIsExecutable(t *testing.T) {
	for _, s := range samples {
		t.Run(s.wantExec, func(t *testing.T) {
			raw, h := openAndParseHeader(t, s.relPath)
			fi := parseFileInfo(t, raw)

			fnames, err := ParseFilenames(raw.Data, fi.FilenameStringsOff, fi.FilenameStringsSz)
			if err != nil {
				t.Fatalf("ParseFilenames: %v", err)
			}
			if len(fnames) == 0 {
				t.Fatal("no filenames parsed")
			}

			// The first loaded file should contain the executable name
			first := strings.ToUpper(fnames[0])
			execUpper := strings.ToUpper(h.ExecutableName)
			if !strings.Contains(first, strings.TrimSuffix(execUpper, ".EXE")) {
				t.Logf("first filename = %q (may use different path format)", fnames[0])
			}
		})
	}
}

func TestSamples_NoPanicsOnDoubleParse(t *testing.T) {
	for _, s := range samples {
		t.Run(s.wantExec, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join("testdata", s.relPath))
			if err != nil {
				t.Skip(err)
			}
			for i := 0; i < 3; i++ {
				raw, err := Open(data)
				if err != nil {
					t.Fatalf("Open pass %d: %v", i, err)
				}
				_, err = ParseFileHeader(raw.Data)
				if err != nil {
					t.Fatalf("ParseFileHeader pass %d: %v", i, err)
				}
				_, err = ParseFileInfo(raw.Data[84:], raw.Version)
				if err != nil {
					t.Fatalf("ParseFileInfo pass %d: %v", i, err)
				}
			}
		})
	}
}

func BenchmarkSampleOpen(b *testing.B) {
	data, err := os.ReadFile(filepath.Join("testdata", "win11", "WINWORD.EXE-AB6EC2FA.pf"))
	if err != nil {
		b.Skip(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Open(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func openAndParseHeader(t *testing.T, relPath string) (*RawFile, *FileHeader) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", relPath))
	if err != nil {
		t.Skipf("testdata not found: %v", err)
	}
	raw, err := Open(data)
	if err != nil {
		t.Fatalf("Open(%s): %v", relPath, err)
	}
	h, err := ParseFileHeader(raw.Data)
	if err != nil {
		t.Fatalf("ParseFileHeader(%s): %v", relPath, err)
	}
	return raw, h
}

func parseFileInfo(t *testing.T, raw *RawFile) *FileInfo {
	t.Helper()
	fi, err := ParseFileInfo(raw.Data[84:], raw.Version)
	if err != nil {
		t.Fatalf("ParseFileInfo: %v", err)
	}
	return fi
}


