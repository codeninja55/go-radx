package command

import (
	"fmt"
	"os"

	"github.com/codeninja55/go-radx/cmd/radx/internal/exitcode"
)

// resolveDICOMPaths expands positional file/directory inputs into the concrete files to process,
// shared by the commands that take a list of DICOM paths (store, modify). A directory is descended
// for *.dcm files only when recursive is set; without it a directory is a usage error. A regular
// file passes through unchanged so a later read surfaces its own per-file error rather than failing
// the whole invocation here. A path that cannot be stat'd also passes through, so the per-file read
// reports the file-I/O fault with the file named.
func resolveDICOMPaths(paths []string, recursive bool) ([]string, error) {
	var out []string
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			out = append(out, p)
			continue
		}
		if !info.IsDir() {
			out = append(out, p)
			continue
		}
		if !recursive {
			return nil, &exitcode.UsageErr{Message: fmt.Sprintf("%s is a directory; pass -R/--recursive to descend into it", p)}
		}
		found, walkErr := dicomFilesUnder(p)
		if walkErr != nil {
			return nil, walkErr
		}
		out = append(out, found...)
	}
	return out, nil
}
