# Format Notes

Working notes on version-specific offsets and quirks for the Windows Prefetch format.

## References

- libyal/libscca — "Windows Prefetch File (PF) format"
- Velocidex/go-prefetch — prior pure-Go implementation

## Versions

| Version | Name     | Compressed |
|---------|----------|------------|
| 17      | Windows XP / Server 2003 | No |
| 23      | Windows Vista / 7        | No |
| 26      | Windows 8                | No |
| 30      | Windows 10               | Yes (MAM + LZXpress Huffman) |
| 31      | Windows 11               | Yes (MAM + LZXpress Huffman) |

## Volume Information — Version Deltas

### Volume entry size

| Version | Entry size |
|---------|------------|
| 17      | 40 bytes   |
| 23, 26  | 104 bytes  |
| 30, 31  | 96 bytes   |

All versions share the same first 36 bytes (device path offset, chars, creation time, serial number, file refs offset/size, dir strings offset/count). The trailing bytes after offset 36 contain unknown/padding and differ per version.

### File references sub-header

| Version | Header size | Header format |
|---------|-------------|---------------|
| 17      | 8 bytes     | 4B unknown (value=1) + 4B count |
| 23+     | 16 bytes    | 4B unknown (value=3) + 4B count + 8B unknown |

References are 8 bytes each (NTFS: 6B MFT entry + 2B sequence number). Sequence number is often 0 on Win8+.

### Directory strings

Each entry: 2-byte character count (LE) + UTF-16LE string + null terminator.

## Open Questions / TBD

- [ ] Version 30 variant 1 vs 2 detection from file info metrics offset
- [ ] Hash string section purpose
- [ ] Compression flag byte semantics (byte 3 of MAM header)
- [ ] v30/v31 trace chain entry differences beyond size
