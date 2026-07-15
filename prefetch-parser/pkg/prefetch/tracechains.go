package prefetch

import (
	"encoding/binary"
	"fmt"
)

type TraceChainEntry struct {
	NextEntryIndex     int32
	TotalBlockLoads    uint32
	Flags              byte
	SampleDurationMs   byte
	BlockUsedBits      byte
	BlockPrefetchedBits byte
}

func traceChainEntrySize(ver Version) int {
	if ver == Version10 || ver == Version11 {
		return 8
	}
	return 12
}

func ParseTraceChains(raw []byte, ver Version, offset uint32, count uint32) ([]TraceChainEntry, error) {
	entrySize := traceChainEntrySize(ver)
	need := int(offset) + int(count)*entrySize
	if len(raw) < need {
		return nil, fmt.Errorf("prefetch: trace chains require %d bytes, file has %d", need, len(raw))
	}

	entries := make([]TraceChainEntry, count)
	for i := uint32(0); i < count; i++ {
		base := int(offset) + int(i)*entrySize
		e := &entries[i]
		if entrySize == 12 {
			e.NextEntryIndex = int32(binary.LittleEndian.Uint32(raw[base:]))
			e.TotalBlockLoads = binary.LittleEndian.Uint32(raw[base+4:])
			e.Flags = raw[base+8]
			e.SampleDurationMs = raw[base+9]
			e.BlockUsedBits = raw[base+10]
			e.BlockPrefetchedBits = raw[base+11]
		} else {
			e.TotalBlockLoads = binary.LittleEndian.Uint32(raw[base:])
			e.Flags = raw[base+4]
			e.SampleDurationMs = raw[base+5]
			e.BlockUsedBits = raw[base+6]
			e.BlockPrefetchedBits = raw[base+7]
		}
	}
	return entries, nil
}
