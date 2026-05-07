package testutil

import (
	"bufio"
	"encoding/json"
	"strings"
	"testing"
)

type SSERecord struct {
	ID    string
	Event string
	Data  []byte
}

func DecodeSSEData(t *testing.T, record SSERecord, dest any) {
	t.Helper()

	if err := json.Unmarshal(record.Data, dest); err != nil {
		t.Fatalf("json.Unmarshal(sse data) error = %v; data=%s", err, string(record.Data))
	}
}

func ParseSSE(t *testing.T, body string) []SSERecord {
	t.Helper()

	scanner := bufio.NewScanner(strings.NewReader(body))
	records := make([]SSERecord, 0)
	current := SSERecord{}

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			records = append(records, current)
			current = SSERecord{}
			continue
		}

		switch {
		case strings.HasPrefix(line, "id: "):
			current.ID = strings.TrimPrefix(line, "id: ")
		case strings.HasPrefix(line, "event: "):
			current.Event = strings.TrimPrefix(line, "event: ")
		case strings.HasPrefix(line, "data: "):
			if len(current.Data) > 0 {
				current.Data = append(current.Data, '\n')
			}
			current.Data = append(current.Data, []byte(strings.TrimPrefix(line, "data: "))...)
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scanner.Err() = %v", err)
	}
	if current.Event != "" || current.ID != "" || len(current.Data) > 0 {
		records = append(records, current)
	}

	return records
}
