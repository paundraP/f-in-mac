package prefetch

import (
	"encoding/binary"
	"errors"
	"fmt"
)

var (
	ErrUnknownSignature   = errors.New("prefetch: not a recognized SCCA/MAM file")
	ErrUnsupportedVersion = errors.New("prefetch: unrecognized version number")
)

type RawFile struct {
	Version       Version
	Data          []byte
	WasCompressed bool
}

func validateVersion(v Version) error {
	switch v {
	case VersionXP, VersionVista7, Version8, Version10, Version11:
		return nil
	default:
		return fmt.Errorf("%w: got %d", ErrUnsupportedVersion, v)
	}
}

func openUncompressed(raw []byte) (*RawFile, error) {
	ver := Version(binary.LittleEndian.Uint32(raw[0:4]))
	if err := validateVersion(ver); err != nil {
		return nil, err
	}
	return &RawFile{Version: ver, Data: raw}, nil
}

func openCompressed(raw []byte) (*RawFile, error) {
	if len(raw) < 8 {
		return nil, fmt.Errorf("prefetch: MAM header too short: %d bytes", len(raw))
	}
	decompressedSize := binary.LittleEndian.Uint32(raw[4:8])
	const maxDecompressedSize = 10 * 1024 * 1024
	if decompressedSize < 8 || decompressedSize > maxDecompressedSize {
		return nil, fmt.Errorf("prefetch: malformed decompressedSize: %d", decompressedSize)
	}

	decompressed, err := lzxpressHuffmanDecompress(raw[8:], int(decompressedSize))
	if err != nil {
		return nil, fmt.Errorf("prefetch: decompression failed: %w", err)
	}
	if len(decompressed) < 8 || string(decompressed[4:8]) != "SCCA" {
		return nil, fmt.Errorf("prefetch: decompressed payload missing SCCA signature")
	}

	ver := Version(binary.LittleEndian.Uint32(decompressed[0:4]))
	if err := validateVersion(ver); err != nil {
		return nil, err
	}
	return &RawFile{Version: ver, Data: decompressed, WasCompressed: true}, nil
}

func Open(raw []byte) (*RawFile, error) {
	if len(raw) < 8 {
		return nil, ErrUnknownSignature
	}

	switch {
	case string(raw[0:3]) == "MAM":
		return openCompressed(raw)
	case string(raw[4:8]) == "SCCA":
		return openUncompressed(raw)
	default:
		return nil, ErrUnknownSignature
	}
}
