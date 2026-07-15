package prefetch

import (
	"encoding/binary"
	"fmt"
)

type FileHeader struct {
	Version        Version
	FileSize       uint32
	ExecutableName string
	PrefetchHash   uint32
	BootPrefetch   bool
}

type FileInfo struct {
	MetricsArrayOffset uint32
	MetricsCount       uint32
	TraceChainsOffset  uint32
	TraceChainsCount   uint32
	FilenameStringsOff uint32
	FilenameStringsSz  uint32
	VolumesInfoOffset  uint32
	VolumesCount       uint32
	VolumesInfoSize    uint32
	LastRunTimes       []uint64
	RunCount           uint32
	HashStringOffset   uint32
	HashStringSize     uint32
}

func ParseFileHeader(raw []byte) (*FileHeader, error) {
	if len(raw) < 84 {
		return nil, fmt.Errorf("prefetch: header too short: %d bytes", len(raw))
	}
	name := decodeUTF16LE(raw[16:76])
	return &FileHeader{
		Version:        Version(binary.LittleEndian.Uint32(raw[0:4])),
		FileSize:       binary.LittleEndian.Uint32(raw[12:16]),
		ExecutableName: name,
		PrefetchHash:   binary.LittleEndian.Uint32(raw[76:80]),
		BootPrefetch:   raw[80]&0x01 != 0,
	}, nil
}

func fileInfoOffsetsFor(raw []byte, ver Version) (FileInfoOffsets, error) {
	if ver == Version10 || ver == Version11 {
		if len(raw) >= 4 {
			mo := binary.LittleEndian.Uint32(raw[0:4])
			if mo == 304 {
				return V30Variant1, nil
			}
		}
		return V30Variant2, nil
	}
	return GetFileInfoOffsets(ver)
}

func ParseFileInfo(raw []byte, ver Version) (*FileInfo, error) {
	off, err := fileInfoOffsetsFor(raw, ver)
	if err != nil {
		return nil, err
	}
	if len(raw) < off.SectionSize {
		return nil, fmt.Errorf("prefetch: file info section too short for version %d: need %d bytes, got %d", ver, off.SectionSize, len(raw))
	}

	fi := &FileInfo{
		MetricsArrayOffset: binary.LittleEndian.Uint32(raw[off.MetricsArrayOffset : off.MetricsArrayOffset+4]),
		MetricsCount:       binary.LittleEndian.Uint32(raw[off.MetricsCount : off.MetricsCount+4]),
		TraceChainsOffset:  binary.LittleEndian.Uint32(raw[off.TraceChainsOffset : off.TraceChainsOffset+4]),
		TraceChainsCount:   binary.LittleEndian.Uint32(raw[off.TraceChainsCount : off.TraceChainsCount+4]),
		FilenameStringsOff: binary.LittleEndian.Uint32(raw[off.FilenameStringsOff : off.FilenameStringsOff+4]),
		FilenameStringsSz:  binary.LittleEndian.Uint32(raw[off.FilenameStringsSize : off.FilenameStringsSize+4]),
		VolumesInfoOffset:  binary.LittleEndian.Uint32(raw[off.VolumesInfoOffset : off.VolumesInfoOffset+4]),
		VolumesCount:       binary.LittleEndian.Uint32(raw[off.VolumesCount : off.VolumesCount+4]),
		VolumesInfoSize:    binary.LittleEndian.Uint32(raw[off.VolumesInfoSize : off.VolumesInfoSize+4]),
		RunCount:           binary.LittleEndian.Uint32(raw[off.RunCount : off.RunCount+4]),
	}

	for i := 0; i < off.LastRunTimesCount; i++ {
		start := off.LastRunTimes + i*8
		fi.LastRunTimes = append(fi.LastRunTimes, binary.LittleEndian.Uint64(raw[start:start+8]))
	}

	if off.HashStringOffset >= 0 {
		fi.HashStringOffset = binary.LittleEndian.Uint32(raw[off.HashStringOffset : off.HashStringOffset+4])
		fi.HashStringSize = binary.LittleEndian.Uint32(raw[off.HashStringSize : off.HashStringSize+4])
	}

	return fi, nil
}
