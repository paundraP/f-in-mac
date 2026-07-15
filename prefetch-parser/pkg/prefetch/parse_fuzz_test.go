package prefetch

import (
	"os"
	"path/filepath"
	"testing"
)

func FuzzParseFileHeader(f *testing.F) {
	// Seed with valid headers from real samples
	for _, rel := range []string{
		"xp/EXPLORER.EXE-082F38A9.pf",
		"win7/CALC.EXE-77FDF17F.pf",
		"win8/NOTEPAD.EXE-D8414F97.pf",
	} {
		data, err := os.ReadFile(filepath.Join("testdata", rel))
		if err == nil {
			raw, err := Open(data)
			if err == nil && len(raw.Data) >= 84 {
				f.Add(raw.Data[:84])
			}
		}
	}

	f.Add([]byte{})
	f.Add([]byte{0, 0, 0, 17, 'S', 'C', 'C', 'A', 0, 0, 0, 0, 0, 0, 0, 0})

	f.Fuzz(func(t *testing.T, header []byte) {
		h, err := ParseFileHeader(header)
		if err != nil && h != nil {
			t.Errorf("non-nil header with error: %v", err)
		}
	})
}

func FuzzParseFileInfo(f *testing.F) {
	for _, rel := range []string{
		"xp/EXPLORER.EXE-082F38A9.pf",
		"win7/CALC.EXE-77FDF17F.pf",
		"win8/NOTEPAD.EXE-D8414F97.pf",
		"win10/SERVICE.EXE-D640A8AF.pf",
		"win11/DLLHOST.EXE-C4F24392.pf",
	} {
		data, err := os.ReadFile(filepath.Join("testdata", rel))
		if err == nil {
			raw, err := Open(data)
			if err == nil && len(raw.Data) > 84 {
				section := raw.Data[84:]
				f.Add(section, uint32(raw.Version))
			}
		}
	}

	f.Add([]byte{}, uint32(17))
	f.Add([]byte{0, 0, 0, 0}, uint32(30))
	f.Add(make([]byte, 256), uint32(26))

	f.Fuzz(func(t *testing.T, section []byte, ver uint32) {
		v := Version(ver)
		fi, err := ParseFileInfo(section, v)
		if err != nil && fi != nil {
			t.Errorf("non-nil fileinfo with error: %v", err)
		}
		if fi != nil {
			if fi.MetricsCount > 100000 {
				t.Errorf("unreasonable MetricsCount: %d", fi.MetricsCount)
			}
			if fi.TraceChainsCount > 100000 {
				t.Errorf("unreasonable TraceChainsCount: %d", fi.TraceChainsCount)
			}
		}
	})
}

func FuzzParseFilenames(f *testing.F) {
	data, err := os.ReadFile(filepath.Join("testdata", "win10", "SERVICE.EXE-D640A8AF.pf"))
	if err == nil {
		if raw, err := Open(data); err == nil {
			if fi, err := ParseFileInfo(raw.Data[84:], raw.Version); err == nil {
				section := raw.Data[fi.FilenameStringsOff : fi.FilenameStringsOff+fi.FilenameStringsSz]
				f.Add(section, fi.FilenameStringsOff, fi.FilenameStringsSz)
			}
		}
	}

	f.Add([]byte{}, uint32(0), uint32(0))
	f.Add([]byte{0, 0}, uint32(0), uint32(2))
	f.Add([]byte{'A', 0, 0, 0, 'B', 0, 0, 0}, uint32(0), uint32(8))

	f.Fuzz(func(t *testing.T, data []byte, offset, size uint32) {
		names, err := ParseFilenames(data, offset, size)
		if err != nil && names != nil {
			t.Errorf("non-nil names with error: %v", err)
		}
	})
}

func FuzzParseFileMetrics(f *testing.F) {
	for _, rel := range []string{
		"xp/EXPLORER.EXE-082F38A9.pf",
		"win10/SERVICE.EXE-D640A8AF.pf",
	} {
		data, err := os.ReadFile(filepath.Join("testdata", rel))
		if err == nil {
			if raw, err := Open(data); err == nil {
				if fi, err := ParseFileInfo(raw.Data[84:], raw.Version); err == nil {
					section := raw.Data[:fi.MetricsArrayOffset+fi.MetricsCount*32]
					f.Add(section, uint32(raw.Version), fi.MetricsArrayOffset, fi.MetricsCount)
				}
			}
		}
	}

	f.Add(make([]byte, 100), uint32(17), uint32(0), uint32(3))
	f.Add(make([]byte, 100), uint32(30), uint32(0), uint32(2))

	f.Fuzz(func(t *testing.T, data []byte, ver uint32, offset, count uint32) {
		metrics, err := ParseFileMetrics(data, Version(ver), offset, count)
		if err != nil && metrics != nil {
			t.Errorf("non-nil metrics with error: %v", err)
		}
	})
}
