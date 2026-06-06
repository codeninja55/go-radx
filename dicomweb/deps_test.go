package dicomweb_test

import (
	"os/exec"
	"strings"
	"testing"
)

// forbiddenCoreDeps are the heavy cloud-provider SDKs that must stay isolated in the
// dicomweb/auth/{aws,gcp} subpackages. The core dicomweb client authenticates only
// through the lightweight pluggable seam (an http.RoundTripper or an oauth2.TokenSource),
// so a caller who never touches a cloud adapter never pays the SDK's transitive-import
// or binary-size cost. This guard fails the build if a future change leaks either SDK
// into the core import graph.
var forbiddenCoreDeps = []string{
	"github.com/aws/aws-sdk-go-v2",
	"github.com/aws/smithy-go",
	"google.golang.org/api",
	"golang.org/x/oauth2/google",
	"cloud.google.com/go",
}

// TestCoreImportGraphExcludesCloudSDKs asserts the transitive dependency set of the core
// dicomweb package contains neither the AWS SDK nor the Google SDK. It shells out to
// `go list -deps` so the assertion reflects the real compiled import graph rather than a
// hand-maintained list.
func TestCoreImportGraphExcludesCloudSDKs(t *testing.T) {
	out, err := exec.Command("go", "list", "-deps", "github.com/codeninja55/go-radx/dicomweb").Output()
	if err != nil {
		t.Fatalf("go list -deps: %v", err)
	}
	deps := strings.Split(strings.TrimSpace(string(out)), "\n")

	for _, dep := range deps {
		for _, forbidden := range forbiddenCoreDeps {
			if dep == forbidden || strings.HasPrefix(dep, forbidden+"/") {
				t.Errorf("core dicomweb import graph leaked a cloud SDK dependency: %s", dep)
			}
		}
	}
}
