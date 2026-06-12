//go:build interop

// Package hl7peer provides a testcontainers-backed python-hl7 MLLP peer fixture for the HL7 v2.x
// interop gate. It starts a pinned Python container, installs the pinned python-hl7 release, and
// runs an asyncio MLLP listener that acknowledges every inbound message with the library's own
// create_ack — so the gate exercises a foreign MLLP implementation in both roles: the listener
// receives go-radx client frames, and mllp_send (the python-hl7 CLI sender) drives the go-radx
// server through the testcontainers host-access tunnel. It is built only under the interop tag so
// the testcontainers dependency stays out of the default build.
package hl7peer

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/testcontainers/testcontainers-go"
	tcexec "github.com/testcontainers/testcontainers-go/exec"
	"github.com/testcontainers/testcontainers-go/wait"
)

// mllpPort is the container-internal port the python-hl7 asyncio listener binds. testcontainers
// maps it to an ephemeral host port discovered after start.
const mllpPort = "2575/tcp"

// image pins the official Python slim image by immutable digest so an interop run resolves the
// same bytes on every runner. The digest is the 3.13.11-slim multi-arch index; bumping it is a
// deliberate, reviewed change — re-resolve with `docker buildx imagetools inspect
// python:<version>-slim` and update the digest here and in tools/versions together.
const image = "docker.io/library/python:3.13.11-slim@sha256:2b9c9803c6a287cafa0a8c917211dddd23dcd2016f049690ee5219f5d3f1636e"

// readyLogLine is printed by the peer script once the asyncio server is accepting connections;
// the wait strategy blocks on it so a test never races the pip install or the bind.
const readyLogLine = "MLLP peer listening"

// peerScript is the asyncio MLLP listener the container runs: for every inbound framed message it
// parses the message with python-hl7 and replies with the library-built original-mode ACK
// (Message.create_ack), so the acknowledgement go-radx parses is authored entirely by the foreign
// implementation. The synthetic test traffic carries no PHI.
const peerScript = `import asyncio

from hl7.mllp import start_hl7_server


async def handle(reader, writer):
    while not reader.at_eof():
        try:
            message = await reader.readmessage()
        except asyncio.IncompleteReadError:
            break
        writer.writemessage(message.create_ack())
        await writer.drain()
    writer.close()
    await writer.wait_closed()


async def main():
    server = await start_hl7_server(handle, host="0.0.0.0", port=2575)
    async with server:
        print("MLLP peer listening", flush=True)
        await server.serve_forever()


asyncio.run(main())
`

// outboundPath is where SendToHost stages the MLLP-wrapped message inside the container before
// invoking mllp_send on it.
const outboundPath = "/peer/outbound.hl7"

// Container wraps a started python-hl7 MLLP peer and the host-side address its mapped listener
// port resolves to.
type Container struct {
	container testcontainers.Container
	host      string
	port      string
}

// Start launches the python-hl7 peer container and blocks until its MLLP listener is accepting
// connections. hostAccessPort names a host (test-process) TCP port the container may dial back at
// testcontainers.HostInternal — the reverse-direction tunnel SendToHost sends through; pass 0 to
// start a receive-only peer. Per the testcontainers HostAccessPorts contract the host service
// MUST already be listening on hostAccessPort when Start is called (the caller binds the go-radx
// server first); the host-access tunnel rides a testcontainers-managed sshd sidecar whose image
// reference is hardcoded in the pinned testcontainers-go module and recorded in tools/versions
// (mllp-peer.sshd.image), where tools/pin-drift.sh holds the two in lockstep. The pip install
// runs at container start against the exact pinned release, so the wait allows for the index
// round-trip as well as the image pull.
func Start(ctx context.Context, hostAccessPort int) (*Container, error) {
	req := testcontainers.ContainerRequest{
		Image:        image,
		ExposedPorts: []string{mllpPort},
		Files: []testcontainers.ContainerFile{{
			Reader:            strings.NewReader(peerScript),
			ContainerFilePath: "/peer/mllp_peer.py",
			FileMode:          0o644,
		}},
		// The exact `==` pin of python-hl7 (PyPI distribution name `hl7`) is recorded in
		// tools/versions and enforced on this line by tools/pin-drift.sh; bumping it is a
		// deliberate, reviewed change made in both places together. The install is a runtime
		// PyPI dependency the other interop legs avoid, so it retries fail-closed: pip's own
		// --retries covers per-request network blips and the shell loop covers whole-install
		// failures, after which the container exits non-zero and the gate fails loudly. The
		// stronger fix — a digest-pinned prebaked peer image — is deferred deliberately:
		// building and publishing an image is outside this repo's CI scope today.
		Cmd: []string{
			"sh", "-c",
			"for attempt in 1 2 3; do" +
				" pip install --no-cache-dir --retries 5 hl7==0.4.5 && break;" +
				" [ \"$attempt\" = 3 ] && exit 1;" +
				" sleep 5;" +
				" done && exec python /peer/mllp_peer.py",
		},
		WaitingFor: wait.ForLog(readyLogLine).WithStartupTimeout(5 * time.Minute),
	}
	if hostAccessPort > 0 {
		req.HostAccessPorts = []int{hostAccessPort}
	}

	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, fmt.Errorf("start python-hl7 MLLP peer container: %w", err)
	}

	host, err := c.Host(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolve MLLP peer host: %w", err)
	}
	mp, err := c.MappedPort(ctx, mllpPort)
	if err != nil {
		return nil, fmt.Errorf("resolve mapped MLLP port: %w", err)
	}

	return &Container{container: c, host: host, port: mp.Port()}, nil
}

// Stop terminates the container. It is safe to call on a nil container.
func (c *Container) Stop(ctx context.Context) error {
	if c == nil || c.container == nil {
		return nil
	}
	return c.container.Terminate(ctx)
}

// Addr returns the host:port a go-radx MLLP client dials to reach the peer listener.
func (c *Container) Addr() string {
	return fmt.Sprintf("%s:%s", c.host, c.port)
}

// SendToHost sends message (raw HL7 bytes, CR segment separators, no MLLP envelope) from inside
// the container to the host port named at Start, using python-hl7's mllp_send CLI — the foreign
// sender — through the testcontainers host-access tunnel. It returns mllp_send's combined output,
// which includes the raw acknowledgement frame the go-radx server replied with, or an error when
// the send exits non-zero. The staged file carries the MLLP envelope because mllp_send's stream
// reader expects wrapped input; the CLI re-frames the message itself on the socket.
func (c *Container) SendToHost(ctx context.Context, hostPort int, message []byte) (string, error) {
	wrapped := append([]byte{0x0b}, message...)
	wrapped = append(wrapped, 0x1c, 0x0d)
	if err := c.container.CopyToContainer(ctx, wrapped, outboundPath, 0o644); err != nil {
		return "", fmt.Errorf("stage outbound message in peer container: %w", err)
	}

	cmd := []string{
		"mllp_send",
		"--file", outboundPath,
		"--port", fmt.Sprintf("%d", hostPort),
		testcontainers.HostInternal,
	}
	code, reader, err := c.container.Exec(ctx, cmd, tcexec.Multiplexed())
	if err != nil {
		return "", fmt.Errorf("exec mllp_send in peer container: %w", err)
	}
	out, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("read mllp_send output: %w", err)
	}
	if code != 0 {
		return string(out), fmt.Errorf("mllp_send exited %d: %s", code, string(out))
	}
	return string(out), nil
}
