package output

import (
	"fmt"

	"prefetch-parser/pkg/prefetch"
)

type PrefetchRecord struct {
	SourceFile    string
	Executable    string
	Hash          string
	Version       uint32
	FileSize      uint32
	RunCount      uint32
	LastRunTimes  []string
	VolumePath    string
	VolumeSerial  string
	VolumeCreated string
	LoadedFiles   []string
	Directories   []string
	FileRefCount  int
	MetricsCount  int
}

func BuildRecord(sourceFile string, raw *prefetch.RawFile, h *prefetch.FileHeader, fi *prefetch.FileInfo, metrics []prefetch.FileMetricsEntry, filenames []string, vi *prefetch.VolumeInfo) PrefetchRecord {
	r := PrefetchRecord{
		SourceFile:   sourceFile,
		Executable:   h.ExecutableName,
		Hash:         fmt.Sprintf("0x%08X", h.PrefetchHash),
		Version:      uint32(h.Version),
		FileSize:     h.FileSize,
		RunCount:     fi.RunCount,
		LoadedFiles:  filenames,
		MetricsCount: len(metrics),
	}

	times := prefetch.FiletimeToTimes(fi.LastRunTimes)
	r.LastRunTimes = make([]string, len(times))
	for i, t := range times {
		if t.Year() > 1601 {
			r.LastRunTimes[i] = t.Format("2006-01-02 15:04:05")
		} else {
			r.LastRunTimes[i] = ""
		}
	}

	if vi != nil && len(vi.Entries) > 0 {
		e := vi.Entries[0]
		r.VolumePath = e.DevicePath
		r.VolumeSerial = fmt.Sprintf("0x%08X", e.SerialNumber)
		if e.CreationTime > 0 {
			r.VolumeCreated = prefetch.FiletimeToTime(e.CreationTime).Format("2006-01-02 15:04:05")
		}
		r.Directories = e.DirectoryStrings
		r.FileRefCount = len(e.FileReferences)
	}

	return r
}
