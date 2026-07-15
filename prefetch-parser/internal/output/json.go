package output

import (
	"encoding/json"
	"io"
)

func WriteJSON(w io.Writer, records []PrefetchRecord) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(records)
}
