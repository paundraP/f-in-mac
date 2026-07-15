package prefetch

import "fmt"

type Version uint32

const (
	VersionXP     Version = 17
	VersionVista7 Version = 23
	Version8      Version = 26
	Version10     Version = 30
	Version11     Version = 31
)

type FileInfoOffsets struct {
	SectionSize         int
	MetricsArrayOffset  int
	MetricsCount        int
	TraceChainsOffset   int
	TraceChainsCount    int
	FilenameStringsOff  int
	FilenameStringsSize int
	VolumesInfoOffset   int
	VolumesCount        int
	VolumesInfoSize     int
	LastRunTimes        int
	LastRunTimesCount   int
	RunCount            int
	HashStringOffset    int
	HashStringSize      int
}

var (
	V30Variant1 = FileInfoOffsets{
		SectionSize:         220,
		MetricsArrayOffset:  0,
		MetricsCount:        4,
		TraceChainsOffset:   8,
		TraceChainsCount:    12,
		FilenameStringsOff:  16,
		FilenameStringsSize: 20,
		VolumesInfoOffset:   24,
		VolumesCount:        28,
		VolumesInfoSize:     32,
		LastRunTimes:        44,
		LastRunTimesCount:   8,
		RunCount:            124,
		HashStringOffset:    136,
		HashStringSize:      140,
	}

	V30Variant2 = FileInfoOffsets{
		SectionSize:         212,
		MetricsArrayOffset:  0,
		MetricsCount:        4,
		TraceChainsOffset:   8,
		TraceChainsCount:    12,
		FilenameStringsOff:  16,
		FilenameStringsSize: 20,
		VolumesInfoOffset:   24,
		VolumesCount:        28,
		VolumesInfoSize:     32,
		LastRunTimes:        44,
		LastRunTimesCount:   8,
		RunCount:            116,
		HashStringOffset:    128,
		HashStringSize:      132,
	}
)

var fileInfoOffsets = map[Version]FileInfoOffsets{
	VersionXP: {
		SectionSize:         68,
		MetricsArrayOffset:  0,
		MetricsCount:        4,
		TraceChainsOffset:   8,
		TraceChainsCount:    12,
		FilenameStringsOff:  16,
		FilenameStringsSize: 20,
		VolumesInfoOffset:   24,
		VolumesCount:        28,
		VolumesInfoSize:     32,
		LastRunTimes:        36,
		LastRunTimesCount:   1,
		RunCount:            60,
		HashStringOffset:    -1,
		HashStringSize:      -1,
	},
	VersionVista7: {
		SectionSize:         156,
		MetricsArrayOffset:  0,
		MetricsCount:        4,
		TraceChainsOffset:   8,
		TraceChainsCount:    12,
		FilenameStringsOff:  16,
		FilenameStringsSize: 20,
		VolumesInfoOffset:   24,
		VolumesCount:        28,
		VolumesInfoSize:     32,
		LastRunTimes:        44,
		LastRunTimesCount:   1,
		RunCount:            68,
		HashStringOffset:    -1,
		HashStringSize:      -1,
	},
	Version8: {
		SectionSize:         220,
		MetricsArrayOffset:  0,
		MetricsCount:        4,
		TraceChainsOffset:   8,
		TraceChainsCount:    12,
		FilenameStringsOff:  16,
		FilenameStringsSize: 20,
		VolumesInfoOffset:   24,
		VolumesCount:        28,
		VolumesInfoSize:     32,
		LastRunTimes:        44,
		LastRunTimesCount:   8,
		RunCount:            124,
		HashStringOffset:    -1,
		HashStringSize:      -1,
	},
}

func GetFileInfoOffsets(ver Version) (FileInfoOffsets, error) {
	off, ok := fileInfoOffsets[ver]
	if !ok {
		return FileInfoOffsets{}, fmt.Errorf("%w: got %d", ErrUnsupportedVersion, ver)
	}
	return off, nil
}
