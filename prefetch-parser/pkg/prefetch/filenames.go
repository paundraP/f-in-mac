package prefetch

import (
	"encoding/binary"
	"fmt"
	"unicode/utf16"
)

func decodeUTF16LE(b []byte) string {
	u16 := make([]uint16, 0, len(b)/2)
	for i := 0; i+1 < len(b); i += 2 {
		u16 = append(u16, binary.LittleEndian.Uint16(b[i:i+2]))
	}
	runes := utf16.Decode(u16)
	for i, r := range runes {
		if r == 0 {
			return string(runes[:i])
		}
	}
	return string(runes)
}

func ParseFilenames(raw []byte, offset uint32, size uint32) ([]string, error) {
	start := int(offset)
	end := start + int(size)
	if start > len(raw) || end > len(raw) {
		return nil, fmt.Errorf("prefetch: filename strings section [%d:%d] exceeds file size %d", start, end, len(raw))
	}

	section := raw[start:end]
	var names []string

	i := 0
	for i < len(section) {
		if i+1 >= len(section) {
			break
		}
		if section[i] == 0 && section[i+1] == 0 {
			i += 2
			continue
		}
		startStr := i
		for i+1 < len(section) && !(section[i] == 0 && section[i+1] == 0) {
			i += 2
		}
		name := decodeUTF16LE(section[startStr:i])
		if name != "" {
			names = append(names, name)
		}
		if i < len(section) {
			i += 2
		}
	}

	return names, nil
}
