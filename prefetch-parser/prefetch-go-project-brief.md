# prefetch-go — Project Brief

A cross-platform (macOS/Linux-first) Windows Prefetch (.pf) file parser written in Go, built as a native reimplementation inspired by Eric Zimmerman's `PECmd`/`Prefetch` — with no dependency on Windows APIs.

**Reference sources**

- Format spec: libyal/libscca — *"Windows Prefetch File (PF) format"* (most precise open spec, versions 17/23/26/30/31)
- Prior art to study (not copy): `Velocidex/go-prefetch` — pure-Go parser with its own LZXpress Huffman decompressor
- Output shape target: PECmd's CSV/JSON field names, for compatibility with existing DFIR timeline tooling

---

## Project Structure

```
prefetch-go/
├── cmd/
│   └── pfparse/
│       └── main.go              # CLI entrypoint only — flags, wiring, no parsing logic
│
├── pkg/
│   └── prefetch/
│       ├── prefetch.go          # Epic 1: Open(), RawFile, signature/version detection
│       ├── version.go           # Epic 3/7: Version enum, per-version offset tables
│       ├── lzxpress.go          # Epic 2: LZXpress Huffman decompressor
│       ├── lzxpress_test.go
│       ├── header.go            # Epic 3: fixed file header + file information section
│       ├── metrics.go           # Epic 4: file metrics array
│       ├── tracechains.go       # Epic 5: trace chains array
│       ├── filenames.go         # Epic 6: filename strings (UTF-16LE)
│       ├── volumes.go           # Epic 7: volume information, directory strings
│       ├── timestamps.go        # Epic 8: FILETIME helpers, last-run-times(8)
│       ├── prefetch_test.go     # integration tests: full file in → struct out
│       └── testdata/
│           ├── xp/*.pf          # v17
│           ├── win7/*.pf        # v23
│           ├── win8/*.pf        # v26
│           ├── win10/*.pf       # v30
│           └── win11/*.pf       # v31
│
├── internal/
│   └── output/
│       ├── csv.go               # Epic 9: PECmd-compatible CSV writer
│       └── json.go              # Epic 9: JSON writer
│
├── docs/
│   └── format-notes.md          # working notes on offsets/quirks per version
│
├── go.mod
├── go.sum
├── README.md
└── Makefile                     # build, test, fuzz, cross-compile targets
```

**Key principles**

- `pkg/prefetch` is a reusable library; `cmd/pfparse` is thin plumbing only.
- One file per format section — each epic maps to one file, one PR.
- `version.go` centralizes per-version offset tables so no file has version if/else sprawl.
- `testdata/` holds real samples per OS version; validated against real PECmd output.

---

## Epic Briefs

### Epic 0 — Project Scaffolding (not yet)

**Goal:** A buildable, testable Go module with the structure above in place.
**Deliverables:** `go.mod`, empty package files with doc comments, `Makefile` (`build`, `test`, `fuzz`, `cross-compile` targets), CI stub (`go vet`, `go test ./...`).
**Depends on:** Nothing.
**Effort:** 0.5 day.

### Epic 1 — File I/O & Version/Compression Detection (on-going)

**Goal:** Given raw bytes, detect `MAM` vs `SCCA` signature, return version number and a routing point to either the decompressor or straight to header parsing.
**Deliverables:** `prefetch.go` — `Open([]byte) (*RawFile, error)`, `Version` enum, signature validation, size-sanity guard against malformed `decompressedSize`.
**Depends on:** Epic 0.
**Effort:** 1 day.

### Epic 2 — LZXpress Huffman Decompressor

**Goal:** Pure-Go decompression of Win10/11 MAM-compressed payloads — the highest-risk piece of the whole project.
**Deliverables:** `lzxpress.go` with chunked decompression, prefix code tree, bitstream reader; independent unit tests with known compressed/decompressed byte pairs; fuzz target.
**Depends on:** Epic 1 (consumes its stubbed function signature).
**Effort:** 3–5 days (hardest epic — budget the most slack here).

### Epic 3 — Header & File Information Parsing

**Goal:** Parse the fixed file header (signature, version, executable name, hash, file size) and the version-dependent file information section (offsets/counts for every other section).
**Deliverables:** `header.go`, `version.go` offset tables for v17/v23/v26 validated against real samples; v30/v31 tables written from spec but marked unvalidated pending Epic 2.
**Depends on:** Epic 1. (Can proceed on v17/v23/v26 *before* Epic 2 is finished — recommended build order.)
**Effort:** 3 days.

### Epic 4 — File Metrics Array

**Goal:** Parse per-file load metrics referenced by trace chains.
**Deliverables:** `metrics.go`, tests against real samples.
**Depends on:** Epic 3.
**Effort:** 1 day.

### Epic 5 — Trace Chains Array

**Goal:** Parse legacy trace chain entries (mostly zeroed on modern Windows, still needed for older versions).
**Deliverables:** `tracechains.go`.
**Depends on:** Epic 3.
**Effort:** 0.5 day.

### Epic 6 — Filename Strings Section

**Goal:** Parse UTF-16LE null-terminated filename strings, offset/count driven from file info.
**Deliverables:** `filenames.go`.
**Depends on:** Epic 3.
**Effort:** 1 day.

### Epic 7 — Volume Information

**Goal:** Parse device paths, volume creation time, serial number, file references (MFT entry/sequence), directory strings — the section with the most version drift.
**Deliverables:** `volumes.go`; per-version deltas documented in `docs/format-notes.md`.
**Depends on:** Epic 3.
**Effort:** 3 days (second-hardest epic after decompression).

### Epic 8 — Timestamp Handling

**Goal:** FILETIME → `time.Time` conversion; correct handling of single last-run-time (v17–26) vs 8-entry last-run-times array (v30+).
**Deliverables:** `timestamps.go`, decision on always-a-slice representation in the public struct.
**Depends on:** Epic 3.
**Effort:** 1 day.

### Epic 9 — Output Layer (PECmd-Compatible)

**Goal:** Flatten the nested internal model into PECmd-shaped records; write CSV and JSON.
**Deliverables:** `internal/output/csv.go`, `internal/output/json.go`, a `ToRecord()` method on the parsed file type.
**Depends on:** Epics 4–8.
**Effort:** 2 days.

### Epic 10 — CLI

**Goal:** Usable command-line tool: single file, directory/batch mode, format flags, resilient handling of corrupt/partial files.
**Deliverables:** `cmd/pfparse/main.go` using `cobra` or stdlib `flag`.
**Depends on:** Epic 9.
**Effort:** 2 days.

### Epic 11 — Testing & Validation

**Goal:** Confidence that output matches ground truth.
**Deliverables:** Cross-checks against real PECmd output (Windows/Wine) on the same samples; fuzz coverage on decompressor and header parsing; ongoing, not a one-time pass.
**Depends on:** Runs alongside every epic from Epic 2 onward.
**Effort:** Ongoing, ~20% overhead on top of each epic.

### Epic 12 — macOS Packaging & Distribution

**Goal:** Easy install for macOS DFIR users.
**Deliverables:** Static cross-compiled binaries (darwin/arm64, darwin/amd64) via GoReleaser, Homebrew tap.
**Depends on:** Epic 10 stable.
**Effort:** 1–2 days.

---

## Timeline (solo, part-time pace)

Assumes evenings/weekends effort, not full-time — adjust the multiplier if you have more dedicated time.

| Phase                        | Epics                | Est. Duration | Cumulative |
| ---------------------------- | -------------------- | ------------- | ---------- |
| 1. Foundations               | 0, 1                 | 1.5 days      | Week 1     |
| 2. Uncompressed path first   | 3 (v17/v23/v26 only) | 3 days        | Week 2     |
| 3. Decompression             | 2                    | 3–5 days     | Week 3–4  |
| 4. Unlock v30/v31            | 3 (validate)         | 1 day         | Week 4     |
| 5. Remaining sections        | 4, 5, 6, 8           | 3.5 days      | Week 5     |
| 6. Hardest remaining section | 7                    | 3 days        | Week 6     |
| 7. Output & CLI              | 9, 10                | 4 days        | Week 7     |
| 8. Validation pass           | 11 (dedicated pass)  | 2–3 days     | Week 8     |
| 9. Packaging & release       | 12                   | 1–2 days     | Week 8–9  |

**Rough total: 7–9 weeks part-time** to a first solid `v1.0` covering all five prefetch versions with validated output. Epic 11 testing runs throughout, not just at the end — the dedicated pass at the end is for full cross-version regression, not first-time testing.

**Critical path:** Epic 2 (decompressor) blocks full v30/v31 support and is the biggest schedule risk — if it slips, everything downstream for those two versions slips with it. Everything else can proceed in parallel against v17/v23/v26 samples in the meantime, which is why the build order front-loads the uncompressed path.
