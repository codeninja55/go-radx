package ui

import (
	"fmt"
	"io"
	"os"

	"github.com/alexeyco/simpletable"
)

// NewTable creates a new simpletable with go-radx styling.
func NewTable() *simpletable.Table {
	table := simpletable.New()
	table.SetStyle(simpletable.StyleCompactLite)
	return table
}

// NewDICOMTagTable creates a table for displaying DICOM tags.
// Columns: Tag, VR, Name, Value
func NewDICOMTagTable() *simpletable.Table {
	table := NewTable()
	table.Header = &simpletable.Header{
		Cells: []*simpletable.Cell{
			{Align: simpletable.AlignCenter, Text: "Tag"},
			{Align: simpletable.AlignCenter, Text: "VR"},
			{Align: simpletable.AlignLeft, Text: "Name"},
			{Align: simpletable.AlignLeft, Text: "Value"},
		},
	}
	return table
}

// AddDICOMTagRow adds a row to a DICOM tag table.
func AddDICOMTagRow(table *simpletable.Table, tag, vr, name, value string) {
	table.Body.Cells = append(table.Body.Cells, []*simpletable.Cell{
		{Text: tag},
		{Text: vr},
		{Text: name},
		{Text: value},
	})
}

// PrintTable prints a table to the given writer (defaults to stdout).
func PrintTable(table *simpletable.Table, w io.Writer) {
	if w == nil {
		w = os.Stdout
	}
	_, _ = fmt.Fprintln(w, table.String())
}
