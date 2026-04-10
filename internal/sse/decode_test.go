package sse

import (
	"context"
	"io"
	"strings"
	"testing"
)

func TestDecodeRejectsNilArguments(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		ctx     context.Context
		body    io.Reader
		handler Handler
		wantErr string
	}{
		{
			name:    "Should reject nil context",
			ctx:     nil,
			body:    strings.NewReader("event: ping\n\n"),
			handler: func(Event) error { return nil },
			wantErr: "sse: context is required",
		},
		{
			name:    "Should reject nil body",
			ctx:     context.Background(),
			body:    nil,
			handler: func(Event) error { return nil },
			wantErr: "sse: body is required",
		},
		{
			name:    "Should reject nil handler",
			ctx:     context.Background(),
			body:    strings.NewReader("event: ping\n\n"),
			handler: nil,
			wantErr: "sse: handler is required",
		},
	}

	for _, tt := range testCases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := Decode(tt.ctx, tt.body, tt.handler)
			if err == nil || err.Error() != tt.wantErr {
				t.Fatalf("Decode() error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}
