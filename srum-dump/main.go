// srumgo - a native macOS/Linux/Windows replacement for srum-dump.py's core
// extraction logic, built on the pure-Go ESE parser github.com/Velocidex/go-ese
// (no cgo, no Windows-only wheels required).
//
// It parses SRUDB.DAT, resolves the SruDbIdMapTable (AppId/UserId -> app
// path or SID), and writes one CSV per known SRUM table.
package main

import (
	"encoding/csv"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode/utf16"

	"github.com/Velocidex/ordereddict"
	"www.velocidex.com/golang/go-ese/parser"
)

// knownTables maps the SRUM extension GUID (as stored in the ESE catalog)
// to the human-friendly table name. Sourced from the public SRUM extension
// GUID list used by community DFIR tooling (e.g. MarkBaggett/srum-dump).
var knownTables = map[string]string{
	"{D10CA2FE-6FCF-4F6D-848E-B2E99266FA89}": "Application Resource Usage",
	"{5C8CF1C7-7257-4F13-B223-970EF5939312}": "App Timeline Provider",
	"{DA73FB89-2BEA-4DDC-86B8-6E048C6DA477}": "Energy Estimator Provider",
	"{FEE4E14F-02A9-4550-B5CE-5FA2DA202E37}": "Energy Usage",
	"{FEE4E14F-02A9-4550-B5CE-5FA2DA202E37}LT": "Energy Usage Long Term",
	"{DD6636C4-8929-4683-974E-22C046A43763}": "Network Connectivity Usage",
	"{973F5D5C-1D90-4944-BE8E-24B94231A174}": "Network Data Usage",
	"{D10CA2FE-6FCF-4F6D-848E-B2E99266FA86}": "Push Notifications",
	"{DC3D3B50-BB90-5066-FA4E-A5F90DD8B677}": "SDP CPU",
	"{CDF8EBF6-7C0F-5AC2-158F-DBFBEE981152}": "SDP Event Log",
	"{EEE2F477-0659-5C47-EF03-6D6BEFD441B3}": "SDP Network",
	"{38AD6548-9313-58F8-45C7-D293BAFDC879}": "SDP Performance Counter",
	"{841A7317-3805-518B-C2EA-AD224CB4AF84}": "SDP Physical Disk",
	"{17F4D97B-F26A-5E79-3A82-90040A47D13D}": "SDP Volume Provider",
	"{7ACBBAA3-D029-4BE4-9A7A-0885927F1D8F}": "VFU",
	"{B6D82AF1-F780-4E17-8077-6CB9AD8A6FC4}": "Tagged Energy Provider",
	"{97C2CE28-A37B-4920-B1E9-8B76CD341EC5}": "Undocumented Windows 10 VM info",
}

// A short set of well-known Windows SIDs for readability. Extend as needed.
var knownSIDs = map[string]string{
	"S-1-5-18": "Local System",
	"S-1-5-19": "Local Service",
	"S-1-5-20": "Network Service",
	"S-1-5-32-544": "Administrators",
	"S-1-5-32-545": "Users",
	"S-1-5-32-546": "Guests",
}

func main() {
	inFile := flag.String("in", "", "Path to SRUDB.DAT")
	outDir := flag.String("out", "srum_output", "Directory to write CSVs to")
	flag.Parse()

	if *inFile == "" {
		fmt.Println("usage: srumgo -in /path/to/SRUDB.DAT -out ./srum_output")
		os.Exit(1)
	}

	f, err := os.Open(*inFile)
	if err != nil {
		log.Fatalf("opening %s: %v", *inFile, err)
	}
	defer f.Close()

	ctx, err := parser.NewESEContext(f)
	if err != nil {
		log.Fatalf("parsing ESE header (is this really an ESE/SRUDB.DAT file?): %v", err)
	}

	catalog, err := parser.ReadCatalog(ctx)
	if err != nil {
		log.Fatalf("reading catalog: %v", err)
	}

	idLookup := buildIdLookup(catalog)
	fmt.Printf("Resolved %d entries from SruDbIdMapTable\n", len(idLookup))

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		log.Fatalf("creating output dir: %v", err)
	}

	tables := catalog.Tables.Keys()
	sort.Strings(tables)

	for _, guid := range tables {
		friendly, known := knownTables[guid]
		if !known {
			continue // skip MSysObjects*, SruDbIdMapTable, SruDbCheckpointTable, unknown tables
		}
		outPath := filepath.Join(*outDir, sanitizeFilename(friendly)+".csv")
		n, err := dumpTableToCSV(catalog, guid, outPath, idLookup)
		if err != nil {
			log.Printf("WARNING: table %s (%s): %v", friendly, guid, err)
			continue
		}
		fmt.Printf("%-32s %-40s -> %5d rows -> %s\n", friendly, guid, n, outPath)
	}
}

// buildIdLookup parses SruDbIdMapTable into IdIndex -> readable string.
// IdType 3 entries are binary SIDs; everything else go-ese already decodes
// to a readable UTF-16 string for us.
func buildIdLookup(catalog *parser.Catalog) map[int32]string {
	lookup := map[int32]string{}

	_ = catalog.DumpTable("SruDbIdMapTable", func(row *ordereddict.Dict) error {
		idType, _ := row.Get("IdType")
		idIndexAny, ok := row.Get("IdIndex")
		if !ok {
			return nil
		}
		idIndex, _ := idIndexAny.(int32)

		blobAny, hasBlob := row.Get("IdBlob")
		blobHex, _ := blobAny.(string)

		if !hasBlob || blobHex == "" {
			lookup[idIndex] = ""
			return nil
		}

		raw, err := hex.DecodeString(blobHex)
		if err != nil {
			// Not hex (shouldn't normally happen) - use as-is.
			lookup[idIndex] = blobHex
			return nil
		}

		if t, ok := idType.(uint8); ok && t == 3 {
			lookup[idIndex] = binarySIDToString(raw)
			return nil
		}

		// Non-SID blobs (app paths, package IDs) are stored UTF-16LE.
		lookup[idIndex] = utf16leToString(raw)
		return nil
	})

	return lookup
}

// binarySIDToString converts a raw Windows SID byte blob into its
// "S-1-5-21-..." string form, per the standard SID binary layout:
// byte0=revision, byte1=subauthority count, bytes2-7=authority (big endian),
// then 4-byte little-endian subauthorities.
func binarySIDToString(sid []byte) string {
	if len(sid) < 8 {
		return hex.EncodeToString(sid)
	}
	revision := sid[0]
	subCount := int(sid[1])

	var authority uint64
	for i := 2; i < 8; i++ {
		authority = (authority << 8) | uint64(sid[i])
	}

	parts := []string{"S", strconv.Itoa(int(revision)), strconv.FormatUint(authority, 10)}

	offset := 8
	for i := 0; i < subCount && offset+4 <= len(sid); i++ {
		sub := uint32(sid[offset]) | uint32(sid[offset+1])<<8 |
			uint32(sid[offset+2])<<16 | uint32(sid[offset+3])<<24
		parts = append(parts, strconv.FormatUint(uint64(sub), 10))
		offset += 4
	}

	sidStr := strings.Join(parts, "-")
	if name, ok := knownSIDs[sidStr]; ok {
		return fmt.Sprintf("%s (%s)", sidStr, name)
	}
	return sidStr
}

// dumpTableToCSV streams every row of an SRUM table to a CSV file,
// resolving AppId/UserId columns against idLookup and formatting
// TimeStamp columns as RFC3339.
func dumpTableToCSV(catalog *parser.Catalog, guid, outPath string, idLookup map[int32]string) (int, error) {
	out, err := os.Create(outPath)
	if err != nil {
		return 0, err
	}
	defer out.Close()

	w := csv.NewWriter(out)
	defer w.Flush()

	var header []string
	rowCount := 0

	err = catalog.DumpTable(guid, func(row *ordereddict.Dict) error {
		if header == nil {
			header = append([]string{}, row.Keys()...)
			if err := w.Write(header); err != nil {
				return err
			}
		}
		record := make([]string, len(header))
		for i, col := range header {
			v, _ := row.Get(col)
			record[i] = formatValue(col, v, idLookup)
		}
		rowCount++
		return w.Write(record)
	})

	return rowCount, err
}

func formatValue(col string, v interface{}, idLookup map[int32]string) string {
	if v == nil {
		return ""
	}

	if (col == "AppId" || col == "UserId") {
		if idx, ok := v.(int32); ok {
			if resolved, found := idLookup[idx]; found && resolved != "" {
				return resolved
			}
		}
	}

	switch val := v.(type) {
	case []byte:
		return hex.EncodeToString(val)
	case fmt.Stringer:
		return val.String()
	default:
		return fmt.Sprintf("%v", val)
	}
}

// utf16leToString decodes a raw UTF-16LE byte blob (as stored by Windows
// for SruDbIdMapTable app-path/package-id entries) into a Go string,
// stopping at the first NUL terminator.
func utf16leToString(raw []byte) string {
	if len(raw)%2 != 0 {
		raw = raw[:len(raw)-1]
	}
	u16 := make([]uint16, 0, len(raw)/2)
	for i := 0; i+1 < len(raw); i += 2 {
		v := uint16(raw[i]) | uint16(raw[i+1])<<8
		if v == 0 {
			break
		}
		u16 = append(u16, v)
	}
	return string(utf16.Decode(u16))
}

func sanitizeFilename(name string) string {
	replacer := strings.NewReplacer(" ", "_", "/", "-")
	return replacer.Replace(name)
}
