package report

import (
	"encoding/json"
	"io"
)

// jsonIndent is the indentation of the json document: two spaces, the shape
// most tools and diffs expect.
const jsonIndent = "  "

// renderJSON writes the document as indented JSON followed by a newline.
// The schema is Report: object keys are fixed, numbers stay numbers, and a
// unit's metrics are an ordered array rather than a map, so the reader sees
// them in the canonical order of config.Metrics().
func renderJSON(w io.Writer, doc Report) error {
	data, err := json.MarshalIndent(doc, "", jsonIndent)
	if err != nil {
		return err
	}
	if _, err := w.Write(data); err != nil {
		return err
	}
	_, err = io.WriteString(w, "\n")
	return err
}
