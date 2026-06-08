package command

import (
	"fmt"
	"net"
	"strconv"

	"github.com/codeninja55/go-radx/cmd/radx/internal/exitcode"
	"github.com/codeninja55/go-radx/dimse"
)

// parseAETitles validates a calling/called AE-title pair before any dial. An over-long or
// otherwise malformed title is a usage-class fault (exit 2), the same class echo raises, so the
// fault is reported as an invocation error rather than a network failure.
func parseAETitles(calling, called string) (callingAE, calledAE dimse.AETitle, err error) {
	callingAE, parseErr := dimse.ParseAETitle(calling)
	if parseErr != nil {
		return "", "", &exitcode.UsageErr{Message: fmt.Sprintf("invalid --calling-ae: %v", parseErr)}
	}
	calledAE, parseErr = dimse.ParseAETitle(called)
	if parseErr != nil {
		return "", "", &exitcode.UsageErr{Message: fmt.Sprintf("invalid --called-ae: %v", parseErr)}
	}
	return callingAE, calledAE, nil
}

// hostPort joins a host and port into the address dimse.Associate dials. It is the single place
// the CLI builds a DICOM peer address, so every network command formats it identically.
func hostPort(host string, port int) string {
	return net.JoinHostPort(host, strconv.Itoa(port))
}
