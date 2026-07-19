# srumgo

A native macOS/Linux/Windows replacement for the SRUM-extraction logic in
`srum-dump.py`. It doesn't depend on the Windows-only `libesedb_python`
wheel — it uses [Velocidex/go-ese](https://github.com/Velocidex/go-ese), a
pure-Go ESE database parser (built by the Velociraptor DFIR team), so it
compiles and runs directly on your Mac.

It parses `SRUDB.DAT`, resolves the `SruDbIdMapTable` (turning numeric
AppId/UserId references into real process names and SIDs), and writes one
CSV per known SRUM table: Application Resource Usage, App Timeline
Provider, Network Data Usage, Network Connectivity Usage, Energy Usage
(+ Long Term), Push Notifications, and VFU.

## Setup (on your Mac)

```bash
mkdir srumgo && cd srumgo
# copy main.go and go.mod from this folder into srumgo/
go mod tidy      # pulls in go-ese v0.2.0 and ordereddict from GitHub
go build -o srumgo .
```

Requires Go (`brew install go` if you don't have it).

The go.mod pins `go-ese v0.2.0` on purpose — `parser.NewESEContext` changed
signature on the unreleased master branch, so pinning avoids a rebuild
surprise if `go mod tidy` is re-run later.

## Usage

```bash
./srumgo -in /path/to/SRUDB.DAT -out ./srum_output
```

This writes CSVs like `Application_Resource_Usage.csv`,
`Network_Data_Usage.csv`, etc. into `./srum_output`.

## Notes / what it does NOT do (yet)

- It doesn't cross-reference the SOFTWARE registry hive to further resolve
  package SIDs/AppIDs into install locations/publishers the way the full
  GUI srum-dump does — you get the raw resolved AppId string (usually a
  full exe path or service name) and resolved user SIDs.
- No XLSX output/pivot template — CSV only, but that opens fine in Excel,
  Numbers, or any DFIR timeline tool (Timeline Explorer, Elastic, etc.).
- The `known_tables` GUID map and general resolution logic were validated
  against the actual test SRUDB.DAT fixture bundled with go-ese: SID
  resolution, service name resolution (WManSvc, WpnService, DiagTrack...),
  and full binary paths came out correctly in testing.
