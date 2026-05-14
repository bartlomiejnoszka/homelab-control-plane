package output

import (
	"encoding/json"
	"io"
)

// WriteJSON writes a pretty-printed JSON representation of v to w.
// io.Writer is Go's common output abstraction, like writing to any stream.
func WriteJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
