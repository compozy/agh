package daytona

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/pedronauck/agh/internal/testutil"
)

func TestSidecarTransportBinaryAssetsContract(t *testing.T) {
	// not parallel: this test intentionally hides PATH to prove runtime sidecar lookup does not use host tools.
	t.Run("Should resolve embedded Linux sidecar binary without host Go toolchain", func(t *testing.T) {
		t.Setenv("PATH", t.TempDir())

		bootstrap := &fakeTransport{readArchives: [][]byte{[]byte("aarch64\n")}}
		transport := &sidecarTransport{
			logger:    slog.New(slog.DiscardHandler),
			bootstrap: bootstrap,
			binaries:  make(map[string][]byte),
		}

		binary, err := transport.sidecarBinary(testutil.Context(t), sandboxInfo{ID: "sandbox-sidecar"})
		if err != nil {
			t.Fatalf("sidecarBinary() error = %v", err)
		}
		if !bytes.HasPrefix(binary, []byte{0x7f, 'E', 'L', 'F'}) {
			t.Fatalf("sidecarBinary() prefix = %x, want ELF binary", binary[:min(len(binary), 4)])
		}
		if got, want := len(bootstrap.dials), 1; got != want {
			t.Fatalf("bootstrap dials = %d, want %d", got, want)
		}
	})
}
