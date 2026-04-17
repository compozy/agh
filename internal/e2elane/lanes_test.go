package e2elane

import (
	"reflect"
	"regexp"
	"testing"
)

func TestPlanForLaneMapsRuntimeWebCombinedAndNightlySlices(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		lane        Lane
		wantGo      []GoSuite
		wantScripts []ScriptSuite
		wantDaemon  bool
		wantNightly bool
	}{
		{
			name:        "runtime lane keeps daemon and transport parity only",
			lane:        LaneRuntime,
			wantGo:      runtimeGoSuites,
			wantScripts: nil,
		},
		{
			name:        "web lane runs daemon served playwright only",
			lane:        LaneWeb,
			wantGo:      nil,
			wantScripts: daemonServedWebSuites,
			wantDaemon:  true,
		},
		{
			name:        "combined lane joins runtime and browser lanes",
			lane:        LaneCombined,
			wantGo:      runtimeGoSuites,
			wantScripts: daemonServedWebSuites,
			wantDaemon:  true,
		},
		{
			name:        "nightly lane adds credentialed runtime and nightly browser slice",
			lane:        LaneNightly,
			wantGo:      append(append([]GoSuite(nil), runtimeGoSuites...), nightlyGoSuites...),
			wantScripts: append(append([]ScriptSuite(nil), daemonServedWebSuites...), nightlyWebSuites...),
			wantDaemon:  true,
			wantNightly: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			plan, err := PlanForLane(tt.lane)
			if err != nil {
				t.Fatalf("PlanForLane(%q) error = %v", tt.lane, err)
			}

			if plan.Lane != tt.lane {
				t.Fatalf("plan.Lane = %q, want %q", plan.Lane, tt.lane)
			}
			if !reflect.DeepEqual(plan.GoSuites, tt.wantGo) {
				t.Fatalf("plan.GoSuites = %#v, want %#v", plan.GoSuites, tt.wantGo)
			}
			if !reflect.DeepEqual(plan.ScriptSuites, tt.wantScripts) {
				t.Fatalf("plan.ScriptSuites = %#v, want %#v", plan.ScriptSuites, tt.wantScripts)
			}
			if plan.RequiresDaemonServedBrowser != tt.wantDaemon {
				t.Fatalf(
					"plan.RequiresDaemonServedBrowser = %v, want %v",
					plan.RequiresDaemonServedBrowser,
					tt.wantDaemon,
				)
			}
			if plan.IncludesCredentialedNightly != tt.wantNightly {
				t.Fatalf(
					"plan.IncludesCredentialedNightly = %v, want %v",
					plan.IncludesCredentialedNightly,
					tt.wantNightly,
				)
			}
		})
	}
}

func TestPlanForLaneKeepsCredentialedNightlyOutOfPRRequiredEntryPoints(t *testing.T) {
	t.Parallel()

	for _, lane := range []Lane{LaneRuntime, LaneWeb, LaneCombined} {
		t.Run(string(lane), func(t *testing.T) {
			t.Parallel()

			plan, err := PlanForLane(lane)
			if err != nil {
				t.Fatalf("PlanForLane(%q) error = %v", lane, err)
			}

			if plan.IncludesCredentialedNightly {
				t.Fatalf("plan.IncludesCredentialedNightly = true, want false")
			}

			for _, suite := range plan.GoSuites {
				for _, pkg := range suite.Packages {
					if pkg == "./internal/environment/daytona" {
						t.Fatalf("plan.GoSuites unexpectedly included nightly package %q", pkg)
					}
				}
				if suite.Run == NightlyRuntimeE2EPattern {
					t.Fatalf("plan.GoSuites unexpectedly included nightly daemon pattern %q", suite.Run)
				}
			}
		})
	}
}

func TestLanePatternsKeepNightlyDaemonScenariosOutOfDefaultRuntimeLane(t *testing.T) {
	t.Parallel()

	runtimePattern := regexp.MustCompile(RuntimeE2EPattern)
	nightlyPattern := regexp.MustCompile(NightlyRuntimeE2EPattern)

	if runtimePattern.MatchString("TestDaemonNightlyE2EAutomationTaskResumesIntoNetworkChannel") {
		t.Fatal("runtime pattern matched nightly daemon test name, want isolation")
	}
	if !nightlyPattern.MatchString("TestDaemonNightlyE2EAutomationTaskResumesIntoNetworkChannel") {
		t.Fatal("nightly pattern did not match nightly daemon test name")
	}
	if nightlyPattern.MatchString("TestDaemonE2EAutomationTaskBackedJobDelegatesTaskRun") {
		t.Fatal("nightly pattern matched default daemon E2E test name, want isolation")
	}
}

func TestPlanForLaneRejectsUnknownLane(t *testing.T) {
	t.Parallel()

	if _, err := PlanForLane("mystery"); err == nil {
		t.Fatal("PlanForLane() error = nil, want non-nil")
	}
}

func TestPlanForLaneReturnsIndependentGoSuitePackageSlices(t *testing.T) {
	t.Parallel()

	plan, err := PlanForLane(LaneRuntime)
	if err != nil {
		t.Fatalf("PlanForLane(%q) error = %v", LaneRuntime, err)
	}
	if len(plan.GoSuites) == 0 || len(plan.GoSuites[0].Packages) == 0 {
		t.Fatalf("plan.GoSuites = %#v, want at least one package entry", plan.GoSuites)
	}

	plan.GoSuites[0].Packages[0] = "./mutated"

	freshPlan, err := PlanForLane(LaneRuntime)
	if err != nil {
		t.Fatalf("PlanForLane(%q) fresh error = %v", LaneRuntime, err)
	}
	if got, want := freshPlan.GoSuites[0].Packages[0], "./internal/daemon"; got != want {
		t.Fatalf("freshPlan.GoSuites[0].Packages[0] = %q, want %q", got, want)
	}
}
