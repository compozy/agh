package update

import "testing"

func TestCompareVersionsTreatsGitDescribeBuildAsBaseVersion(t *testing.T) {
	t.Run("Should compare git-describe builds using the base tagged version", func(t *testing.T) {
		t.Parallel()

		got, err := compareVersions("v1.2.3-4-gabcdef1", "v1.2.4")
		if err != nil {
			t.Fatalf("compareVersions() error = %v", err)
		}
		if got >= 0 {
			t.Fatalf("compareVersions() = %d, want negative value", got)
		}
	})
}

func TestTrimGitDescribeSuffixLeavesTaggedVersionUntouched(t *testing.T) {
	t.Run("Should leave tagged versions untouched when no git-describe suffix exists", func(t *testing.T) {
		t.Parallel()

		if got := trimGitDescribeSuffix("v1.2.3"); got != "v1.2.3" {
			t.Fatalf("trimGitDescribeSuffix() = %q, want %q", got, "v1.2.3")
		}
	})
}
