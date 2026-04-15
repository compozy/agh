package bridgesdk

import (
	"context"
	"testing"
)

func TestHostAPIClientConstructorsAndCallValidation(t *testing.T) {
	t.Parallel()

	if client := NewHostAPIClient(nil); client != nil {
		t.Fatalf("NewHostAPIClient(nil) = %#v, want nil", client)
	}
	if client := NewHostAPIClientFromCall(nil); client != nil {
		t.Fatalf("NewHostAPIClientFromCall(nil) = %#v, want nil", client)
	}

	client := &HostAPIClient{}
	if err := client.Call(context.Background(), "bridges/instances/list", nil, nil); err == nil {
		t.Fatal("client.Call() error = nil, want non-nil")
	}
}
