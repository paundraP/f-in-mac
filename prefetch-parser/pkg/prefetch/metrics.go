package prefetch

import (
	"encoding/binary"
	"fmt"
)

type FileMetricsEntry struct {
	TraceChainIndex      uint32
	TraceEntriesCount    uint32
	BlocksToPrefetch     uint32
	FilenameStringOffset uint32
	FilenameStringLength uint32
	Flags                uint32
	FileReference        uint64
}

func metricsEntrySize(ver Version) int {
	if ver == VersionXP {
		return 20
	}
	return 32
}

func ParseFileMetrics(raw []byte, ver Version, offset uint32, count uint32) ([]FileMetricsEntry, error) {
	entrySize := metricsEntrySize(ver)
	need := int(offset) + int(count)*entrySize
	if len(raw) < need {
		return nil, fmt.Errorf("prefetch: metrics array requires %d bytes, file has %d", need, len(raw))
	}

	entries := make([]FileMetricsEntry, count)
	for i := uint32(0); i < count; i++ {
		base := int(offset) + int(i)*entrySize
		e := &entries[i]
		e.TraceChainIndex = binary.LittleEndian.Uint32(raw[base:])
		e.TraceEntriesCount = binary.LittleEndian.Uint32(raw[base+4:])
		if ver != VersionXP {
			e.BlocksToPrefetch = binary.LittleEndian.Uint32(raw[base+8:])
			e.FilenameStringOffset = binary.LittleEndian.Uint32(raw[base+12:])
			e.FilenameStringLength = binary.LittleEndian.Uint32(raw[base+16:])
			e.Flags = binary.LittleEndian.Uint32(raw[base+20:])
			e.FileReference = binary.LittleEndian.Uint64(raw[base+24:])
		} else {
			e.FilenameStringOffset = binary.LittleEndian.Uint32(raw[base+8:])
			e.FilenameStringLength = binary.LittleEndian.Uint32(raw[base+12:])
			e.Flags = binary.LittleEndian.Uint32(raw[base+16:])
		}
	}
	return entries, nil
}
