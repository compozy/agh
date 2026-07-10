//go:build mage

package main

import (
	"bytes"
	"runtime"
	"slices"
	"strconv"
	"testing"
)

func TestGoUnitTestPackageLimit(t *testing.T) {
	// t.Setenv forbids t.Parallel (L-002); the whole test stays serial.
	defaultLimit := runtime.NumCPU() / 2
	if defaultLimit < 4 {
		defaultLimit = 4
	}
	cases := []struct {
		name  string
		value string
		want  string
	}{
		{
			name:  "Should default to half the cores floored at four when unset",
			value: "",
			want:  strconv.Itoa(defaultLimit),
		},
		{name: "Should honor a valid override", value: "2", want: "2"},
		{name: "Should ignore a non-numeric override", value: "many", want: strconv.Itoa(defaultLimit)},
		{name: "Should ignore a non-positive override", value: "0", want: strconv.Itoa(defaultLimit)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(goTestPackageLimitEnvVar, tc.value)
			if got := goUnitTestPackageLimit(); got != tc.want {
				t.Fatalf("goUnitTestPackageLimit() with %q = %q, want %q", tc.value, got, tc.want)
			}
		})
	}
}

func TestHermeticGoTestEnvFromBase(t *testing.T) {
	t.Parallel()

	t.Run("Should scrub ambient runtime-state vars and keep everything else", func(t *testing.T) {
		t.Parallel()
		base := []string{
			"AGH_HOME=/tmp/stale-qa-lab/runtime",
			"AGH_HTTP_PORT=54321",
			"AGH_TEST_DAEMON_BIN=/tmp/daemon",
			"AGH_LIVE_DISCOVERY_HELPER=/tmp/helper",
			"PROVIDER_HOME=/tmp/stale-provider",
			"PATH=/usr/bin",
		}
		var log bytes.Buffer
		got := hermeticGoTestEnvFromBase(base, nil, &log)
		for _, banned := range []string{"AGH_HOME=/tmp/stale-qa-lab/runtime", "AGH_HTTP_PORT=54321", "PROVIDER_HOME=/tmp/stale-provider"} {
			if slices.Contains(got, banned) {
				t.Fatalf("hermeticGoTestEnvFromBase() kept ambient %q; env = %v", banned, got)
			}
		}
		for _, kept := range []string{"AGH_TEST_DAEMON_BIN=/tmp/daemon", "AGH_LIVE_DISCOVERY_HELPER=/tmp/helper", "PATH=/usr/bin"} {
			if !slices.Contains(got, kept) {
				t.Fatalf("hermeticGoTestEnvFromBase() dropped %q; env = %v", kept, got)
			}
		}
		if !bytes.Contains(log.Bytes(), []byte("AGH_HOME")) {
			t.Fatalf("scrub note missing dropped var name; log = %q", log.String())
		}
	})

	t.Run("Should let explicit lane overrides win over the scrub", func(t *testing.T) {
		t.Parallel()
		base := []string{"AGH_WEB_DIST_DIR=/tmp/stale-dist"}
		got := hermeticGoTestEnvFromBase(
			base,
			map[string]string{"AGH_WEB_DIST_DIR": "/lane/dist", "CGO_ENABLED": "1"},
			nil,
		)
		if !slices.Contains(got, "AGH_WEB_DIST_DIR=/lane/dist") {
			t.Fatalf("lane override missing; env = %v", got)
		}
		if !slices.Contains(got, "CGO_ENABLED=1") {
			t.Fatalf("added override missing; env = %v", got)
		}
		if slices.Contains(got, "AGH_WEB_DIST_DIR=/tmp/stale-dist") {
			t.Fatalf("stale ambient value survived; env = %v", got)
		}
	})
}

func TestMergeEnvOverrides(t *testing.T) {
	t.Parallel()

	t.Run("Should replace existing entries and append new ones", func(t *testing.T) {
		t.Parallel()
		got := mergeEnvOverrides([]string{"A=1", "B=2"}, map[string]string{"B": "3", "C": "4"})
		for _, want := range []string{"A=1", "B=3", "C=4"} {
			if !slices.Contains(got, want) {
				t.Fatalf("mergeEnvOverrides() = %v, want to contain %q", got, want)
			}
		}
		if slices.Contains(got, "B=2") {
			t.Fatalf("mergeEnvOverrides() kept stale entry; got %v", got)
		}
	})

	t.Run("Should return a copy when no overrides are given", func(t *testing.T) {
		t.Parallel()
		base := []string{"A=1"}
		got := mergeEnvOverrides(base, nil)
		if len(got) != 1 || got[0] != "A=1" {
			t.Fatalf("mergeEnvOverrides() = %v, want [A=1]", got)
		}
	})
}
