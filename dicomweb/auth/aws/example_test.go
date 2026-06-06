package aws_test

import (
	"context"
	"net/http"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	awsauth "github.com/codeninja55/go-radx/dicomweb/auth/aws"
)

// ExampleSigV4RoundTripper shows how a caller builds the SigV4 transport from the AWS
// default config chain and layers it under a standard http.Client, the way a DICOMweb
// client wires it through dicomweb.WithRoundTripper. It is the compiler-verified twin of
// the worked example in docs/user-guide/dicomweb/cloud-auth.md.
func ExampleSigV4RoundTripper() {
	ctx := context.Background()
	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion("us-east-1"))
	if err != nil {
		return
	}

	endpoint := "https://dicom-medical-imaging.us-east-1.amazonaws.com"
	rt, err := awsauth.SigV4RoundTripper(cfg, "us-east-1", endpoint, http.DefaultTransport)
	if err != nil {
		return
	}
	_ = &http.Client{Transport: rt}
	// Output:
}
