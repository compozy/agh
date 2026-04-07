package core

import (
	"io"
	"time"
)

// SSEMessage is the shared SSE envelope.
type SSEMessage struct {
	ID   string
	Name string
	Data any
}

// FlushWriter is an SSE writer that can flush streamed content.
type FlushWriter interface {
	io.Writer
	Flush()
}

// ObserveCursor is the shared cursor used for observe event streaming.
type ObserveCursor struct {
	Timestamp time.Time
	ID        string
}
