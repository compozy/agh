//go:build linux

package daemon

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

func collectResidentMemory() (uint64, string, error) {
	raw, err := os.ReadFile("/proc/self/statm")
	if err != nil {
		return 0, "current", fmt.Errorf("read /proc/self/statm: %w", err)
	}
	fields := strings.Fields(string(raw))
	if len(fields) < 2 {
		return 0, "current", fmt.Errorf("parse /proc/self/statm: expected resident page count")
	}
	residentPages, err := strconv.ParseUint(fields[1], 10, 64)
	if err != nil {
		return 0, "current", fmt.Errorf("parse /proc/self/statm resident pages: %w", err)
	}
	return residentPages * uint64(os.Getpagesize()), "current", nil
}
