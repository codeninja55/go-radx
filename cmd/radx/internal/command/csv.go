package command

import (
	"encoding/csv"

	"github.com/codeninja55/go-radx/cmd/radx/internal/cli"
)

// csvWriter wraps the encoding/csv writer so a command holds one instance across a streamed
// result set, writing rows as they arrive and flushing once at the end. It centralises the
// flush-and-check-error discipline RFC 4180 output needs (a deferred csv.Writer error is silent
// until Error() is read), so a partial CSV write is never reported as a clean run.
type csvWriter struct {
	w *csv.Writer
}

// newCSVWriter returns a csvWriter over the output's machine sink.
func newCSVWriter(out *cli.Output) *csvWriter {
	return &csvWriter{w: out.CSVWriter()}
}

// write emits one row.
func (c *csvWriter) write(row []string) error {
	return c.w.Write(row)
}

// flush flushes buffered rows and returns the first write error the writer accumulated, so a
// truncated or failed CSV write surfaces rather than passing silently.
func (c *csvWriter) flush() error {
	c.w.Flush()
	return c.w.Error()
}
