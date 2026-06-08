module github.com/codeninja55/go-radx/cmd/radx

go 1.26.4

require (
	github.com/alecthomas/kong v1.15.0
	github.com/codeninja55/go-radx v0.0.0-00010101000000-000000000000
	go.uber.org/zap v1.28.0
	modernc.org/sqlite v1.52.0
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/ncruces/go-strftime v1.0.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	go.opentelemetry.io/otel v1.44.0 // indirect
	go.opentelemetry.io/otel/metric v1.44.0 // indirect
	go.opentelemetry.io/otel/trace v1.44.0 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	golang.org/x/oauth2 v0.36.0 // indirect
	golang.org/x/sys v0.45.0 // indirect
	golang.org/x/text v0.34.0 // indirect
	modernc.org/libc v1.72.3 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.11.0 // indirect
)

// The CLI builds against the in-tree library rather than a published version: the
// pseudo-version above is a placeholder the replace resolves to the repository root, so
// the GOWORK=off cmd-radx CI job and a downstream `go build` of this module both compile
// the library source sitting beside it, not a stale module-cache copy.
replace github.com/codeninja55/go-radx => ../..
