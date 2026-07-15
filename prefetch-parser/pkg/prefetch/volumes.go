package prefetch

import (
	"encoding/binary"
	"fmt"
)

type VolumeEntry struct {
	DevicePath       string
	DevicePathChars  uint32
	CreationTime     uint64
	SerialNumber     uint32
	FileReferences   []uint64
	DirectoryStrings []string
}

type VolumeInfo struct {
	Entries []VolumeEntry
}

func volumeEntrySize(ver Version) int {
	switch ver {
	case VersionXP:
		return 40
	case VersionVista7, Version8:
		return 104
	default:
		return 96
	}
}

func fileRefsHeaderSize(ver Version) int {
	if ver == VersionXP {
		return 8
	}
	return 16
}

func ParseVolumeInfo(raw []byte, ver Version, voffset uint32, count uint32, vsize uint32) (*VolumeInfo, error) {
	if int(voffset)+int(vsize) > len(raw) {
		return nil, fmt.Errorf("prefetch: volume info section [%d:%d] exceeds file size %d", voffset, voffset+vsize, len(raw))
	}

	entrySize := volumeEntrySize(ver)
	entriesStart := int(voffset)
	entriesEnd := entriesStart + int(count)*entrySize
	if entriesEnd > int(voffset)+int(vsize) {
		return nil, fmt.Errorf("prefetch: volume info entries require %d bytes, only %d available", int(count)*entrySize, vsize)
	}

	vi := &VolumeInfo{Entries: make([]VolumeEntry, count)}
	for i := uint32(0); i < count; i++ {
		base := entriesStart + int(i)*entrySize
		e := &vi.Entries[i]

		devPathOff := int(voffset) + int(binary.LittleEndian.Uint32(raw[base:]))
		e.DevicePathChars = binary.LittleEndian.Uint32(raw[base+4:])
		e.CreationTime = binary.LittleEndian.Uint64(raw[base+8:])
		e.SerialNumber = binary.LittleEndian.Uint32(raw[base+16:])
		frOff := int(voffset) + int(binary.LittleEndian.Uint32(raw[base+20:]))
		frSize := binary.LittleEndian.Uint32(raw[base+24:])
		dsOff := int(voffset) + int(binary.LittleEndian.Uint32(raw[base+28:]))
		dsCount := binary.LittleEndian.Uint32(raw[base+32:])

		if devPathOff >= 0 && devPathOff < len(raw) {
			e.DevicePath = decodeUTF16LE(raw[devPathOff:])
		}

		if frOff >= 0 && frOff < len(raw) {
			e.FileReferences = parseFileRefs(raw, ver, frOff, frSize)
		}

		if dsOff >= 0 && dsOff < len(raw) {
			e.DirectoryStrings = parseDirStrings(raw, dsOff, dsCount)
		}
	}

	return vi, nil
}

func parseFileRefs(raw []byte, ver Version, off int, size uint32) []uint64 {
	if size == 0 || off < 0 || off+int(size) > len(raw) {
		return nil
	}

	hdrSize := fileRefsHeaderSize(ver)
	if int(size) < hdrSize {
		return nil
	}

	refCount := binary.LittleEndian.Uint32(raw[off+4:])
	refsStart := off + hdrSize
	refsEnd := refsStart + int(refCount)*8
	if refsEnd > off+int(size) {
		refCount = uint32((off + int(size) - refsStart) / 8)
	}

	refs := make([]uint64, refCount)
	for i := uint32(0); i < refCount; i++ {
		refs[i] = binary.LittleEndian.Uint64(raw[refsStart+int(i)*8:])
	}
	return refs
}

func parseDirStrings(raw []byte, off int, count uint32) []string {
	if off < 0 || off > len(raw) {
		return nil
	}
	dirs := make([]string, 0, count)
	pos := off
	for i := uint32(0); i < count; i++ {
		if pos+2 > len(raw) {
			break
		}
		charCount := int(binary.LittleEndian.Uint16(raw[pos:]))
		pos += 2
		strLen := charCount * 2
		if pos+strLen > len(raw) {
			break
		}
		s := decodeUTF16LE(raw[pos : pos+strLen])
		dirs = append(dirs, s)
		pos += strLen
		// skip null terminator
		if pos+2 <= len(raw) && raw[pos] == 0 && raw[pos+1] == 0 {
			pos += 2
		}
	}
	return dirs
}
