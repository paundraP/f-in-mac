package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"prefetch-parser/internal/output"
	"prefetch-parser/pkg/prefetch"
)

func main() {
	filePath := flag.String("file", "", "path to a single .pf file")
	dirPath := flag.String("dir", "", "directory of .pf files to process")
	useJSON := flag.Bool("json", false, "output as JSON (default CSV)")
	detailed := flag.Bool("detailed", false, "detailed CSV with one row per loaded file")
	flag.Parse()

	if *filePath == "" && *dirPath == "" {
		fmt.Fprintln(os.Stderr, "usage: pfparse -file <path> [-json] [-detailed]")
		fmt.Fprintln(os.Stderr, "       pfparse -dir <directory> [-json] [-detailed]")
		os.Exit(1)
	}

	var recs []output.PrefetchRecord

	if *filePath != "" {
		rec, err := processFile(*filePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		recs = append(recs, rec)
	}

	if *dirPath != "" {
		entries, err := os.ReadDir(*dirPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error reading directory: %v\n", err)
			os.Exit(1)
		}
		for _, entry := range entries {
			if entry.IsDir() || !isPrefetchFile(entry.Name()) {
				continue
			}
			path := filepath.Join(*dirPath, entry.Name())
			rec, err := processFile(path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "warning: %s: %v\n", path, err)
				continue
			}
			recs = append(recs, rec)
		}
	}

	if len(recs) == 0 {
		fmt.Fprintln(os.Stderr, "no prefetch files processed")
		os.Exit(1)
	}

	if *useJSON {
		if err := output.WriteJSON(os.Stdout, recs); err != nil {
			fmt.Fprintf(os.Stderr, "error writing JSON: %v\n", err)
			os.Exit(1)
		}
	} else if *detailed {
		if err := output.WriteDetailedCSV(os.Stdout, recs); err != nil {
			fmt.Fprintf(os.Stderr, "error writing CSV: %v\n", err)
			os.Exit(1)
		}
	} else {
		if err := output.WriteCSV(os.Stdout, recs); err != nil {
			fmt.Fprintf(os.Stderr, "error writing CSV: %v\n", err)
			os.Exit(1)
		}
	}
}

func processFile(path string) (output.PrefetchRecord, error) {
	var rec output.PrefetchRecord

	data, err := os.ReadFile(path)
	if err != nil {
		return rec, fmt.Errorf("read: %w", err)
	}

	raw, err := prefetch.Open(data)
	if err != nil {
		return rec, fmt.Errorf("open: %w", err)
	}

	h, err := prefetch.ParseFileHeader(raw.Data)
	if err != nil {
		return rec, fmt.Errorf("header: %w", err)
	}

	fi, err := prefetch.ParseFileInfo(raw.Data[84:], raw.Version)
	if err != nil {
		return rec, fmt.Errorf("fileinfo: %w", err)
	}

	metrics, err := prefetch.ParseFileMetrics(raw.Data, raw.Version, fi.MetricsArrayOffset, fi.MetricsCount)
	if err != nil {
		return rec, fmt.Errorf("metrics: %w", err)
	}

	filenames, err := prefetch.ParseFilenames(raw.Data, fi.FilenameStringsOff, fi.FilenameStringsSz)
	if err != nil {
		return rec, fmt.Errorf("filenames: %w", err)
	}

	var vi *prefetch.VolumeInfo
	if fi.VolumesCount > 0 {
		v, err := prefetch.ParseVolumeInfo(raw.Data, raw.Version, fi.VolumesInfoOffset, fi.VolumesCount, fi.VolumesInfoSize)
		if err != nil {
			return rec, fmt.Errorf("volumes: %w", err)
		}
		vi = v
	}

	rec = output.BuildRecord(path, raw, h, fi, metrics, filenames, vi)
	return rec, nil
}

func isPrefetchFile(name string) bool {
	return strings.HasSuffix(strings.ToLower(name), ".pf")
}
