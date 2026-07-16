# pfparser - Windows Prefetch File Parser

A Go library and CLI tool for parsing Windows Prefetch (`.pf`) files, supporting versions 17 (XP) through 31 (Windows 11), including LZXpress Huffman decompression.

## Install

```sh
# Via install script (copies pre-built binary or builds from source)
./install.sh

# Or build from source
make build
```

Pre-built binaries are in `bin/`:

- `pfparse-darwin-amd64`
- `pfparse-darwin-arm64`
- `pfparse-linux-amd64`

## Usage

```sh
# Parse a single .pf file (CSV output)
pfparse -file path/to/file.pf

# Parse all .pf files in a directory
pfparse -dir path/to/directory/

# JSON output
pfparse -file file.pf -json

# Detailed CSV (one row per loaded file)
pfparse -file file.pf -detailed
```

### CSV columns (default)

| Column          | Description                    |
| --------------- | ------------------------------ |
| SourceFile      | Path to the .pf file           |
| Executable      | Executable name                |
| Hash            | Prefetch hash                  |
| Version         | File format version            |
| FileSize        | File size in bytes             |
| RunCount        | Number of times executed       |
| LastRunTime0..N | Last run timestamps (UTC)      |
| VolumePath      | Volume device path             |
| VolumeSerial    | Volume serial number           |
| VolumeCreated   | Volume creation timestamp      |
| FileRefCount    | File reference count           |
| MetricsCount    | Number of file metrics entries |

## Library

```go
import "prefetch-parser/pkg/prefetch"

data, _ := os.ReadFile("file.pf")
raw, _ := prefetch.Open(data)
h, _ := prefetch.ParseFileHeader(raw.Data)
fi, _ := prefetch.ParseFileInfo(raw.Data[84:], raw.Version)
metrics, _ := prefetch.ParseFileMetrics(raw.Data, raw.Version, fi.MetricsArrayOffset, fi.MetricsCount)
filenames, _ := prefetch.ParseFilenames(raw.Data, fi.FilenameStringsOff, fi.FilenameStringsSz)
vi, _ := prefetch.ParseVolumeInfo(raw.Data, raw.Version, fi.VolumesInfoOffset, fi.VolumesCount, fi.VolumesInfoSize)
```

## Supported versions

| Version | Windows          | Compressed     |
| ------- | ---------------- | -------------- |
| 17      | XP / Server 2003 | No             |
| 23      | Vista / 7        | No             |
| 26      | Windows 8        | No             |
| 30      | Windows 10       | Yes (LZXpress) |
| 31      | Windows 11       | Yes (LZXpress) |

## Build & test

```sh
make build      # build binary
make test       # run tests
make vet        # static analysis
make cross      # cross-compile for all targets
make clean      # remove build artifacts
```

## Format reference

See [docs/format-notes.md](docs/format-notes.md) for detailed format documentation.
