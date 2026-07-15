package output

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
)

func WriteCSV(w io.Writer, records []PrefetchRecord) error {
	maxRuns := 0
	for _, r := range records {
		if len(r.LastRunTimes) > maxRuns {
			maxRuns = len(r.LastRunTimes)
		}
	}

	header := []string{
		"SourceFile", "Executable", "Hash", "Version",
		"FileSize", "RunCount",
	}
	for i := 0; i < maxRuns; i++ {
		header = append(header, fmt.Sprintf("LastRunTime%d", i))
	}
	header = append(header, []string{
		"VolumePath", "VolumeSerial", "VolumeCreated",
		"FileRefCount", "MetricsCount",
	}...)

	cw := csv.NewWriter(w)
	cw.Write(header)

	for _, rec := range records {
		row := []string{
			rec.SourceFile, rec.Executable, rec.Hash,
			strconv.Itoa(int(rec.Version)),
			strconv.Itoa(int(rec.FileSize)),
			strconv.Itoa(int(rec.RunCount)),
		}
		for i := 0; i < maxRuns; i++ {
			if i < len(rec.LastRunTimes) {
				row = append(row, rec.LastRunTimes[i])
			} else {
				row = append(row, "")
			}
		}
		row = append(row, []string{
			rec.VolumePath, rec.VolumeSerial, rec.VolumeCreated,
			strconv.Itoa(rec.FileRefCount),
			strconv.Itoa(rec.MetricsCount),
		}...)

		cw.Write(row)
	}

	cw.Flush()
	return cw.Error()
}

func WriteDetailedCSV(w io.Writer, records []PrefetchRecord) error {
	header := []string{
		"SourceFile", "Executable", "Hash", "Version",
		"LoadedFile", "Directory",
	}

	cw := csv.NewWriter(w)
	cw.Write(header)

	for _, rec := range records {
		maxRows := len(rec.LoadedFiles)
		if len(rec.Directories) > maxRows {
			maxRows = len(rec.Directories)
		}
		if maxRows == 0 {
			maxRows = 1
		}
		for i := 0; i < maxRows; i++ {
			file := ""
			dir := ""
			if i < len(rec.LoadedFiles) {
				file = rec.LoadedFiles[i]
			}
			if i < len(rec.Directories) {
				dir = rec.Directories[i]
			}
			row := []string{rec.SourceFile, rec.Executable, rec.Hash, strconv.Itoa(int(rec.Version)), file, dir}
			cw.Write(row)
		}
	}

	cw.Flush()
	return cw.Error()
}

func loadedFilesString(files []string) string {
	if len(files) > 100 {
		return strings.Join(files[:100], "|") + fmt.Sprintf("... (%d more)", len(files)-100)
	}
	return strings.Join(files, "|")
}
