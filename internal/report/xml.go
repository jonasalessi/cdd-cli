package report

import (
	"encoding/xml"
	"io"
)

// xmlIndent is the indentation of the xml document, two spaces like the
// json one.
const xmlIndent = "  "

// renderXML writes the document as an indented XML tree under the standard
// header. It marshals the same Report the json format does: scalars are
// attributes, lists are elements.
func renderXML(w io.Writer, doc Report) error {
	if _, err := io.WriteString(w, xml.Header); err != nil {
		return err
	}
	data, err := xml.MarshalIndent(doc, "", xmlIndent)
	if err != nil {
		return err
	}
	if _, err := w.Write(data); err != nil {
		return err
	}
	_, err = io.WriteString(w, "\n")
	return err
}
