package store

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestSessionLineageNormalizeAndValidate(t *testing.T) {
	t.Parallel()

	root := NormalizeSessionLineage("sess-root", nil)
	if root == nil || root.RootSessionID != "sess-root" || root.ParentSessionID != "" || root.SpawnDepth != 0 {
		t.Fatalf("NormalizeSessionLineage(root) = %#v", root)
	}
	if err := ValidateSessionLineage("sess-root", root); err != nil {
		t.Fatalf("ValidateSessionLineage(root) error = %v", err)
	}

	ttl := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	child := NormalizeSessionLineage("sess-child", &SessionLineage{
		ParentSessionID: " sess-root ",
		RootSessionID:   " sess-root ",
		SpawnDepth:      1,
		SpawnRole:       " worker ",
		TTLExpiresAt:    &ttl,
		PermissionPolicy: SessionPermissionPolicy{
			Tools: []string{" read ", "edit", "edit"},
		},
	})
	if child.ParentSessionID != "sess-root" ||
		child.RootSessionID != "sess-root" ||
		child.SpawnRole != "worker" ||
		len(child.PermissionPolicy.Tools) != 2 ||
		child.PermissionPolicy.Tools[0] != "edit" ||
		child.PermissionPolicy.Tools[1] != "read" {
		t.Fatalf("NormalizeSessionLineage(child) = %#v", child)
	}
	if err := ValidateSessionLineage("sess-child", child); err != nil {
		t.Fatalf("ValidateSessionLineage(child) error = %v", err)
	}
}

func TestSessionLineageValidationRejectsInvalidPolicyAndBudget(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		id      string
		lineage *SessionLineage
		want    string
	}{
		{
			name: "negative budget",
			id:   "sess-root",
			lineage: &SessionLineage{
				RootSessionID: "sess-root",
				SpawnBudget:   SessionSpawnBudget{MaxChildren: -1},
			},
			want: "max_children",
		},
		{
			name: "empty permission atom",
			id:   "sess-root",
			lineage: &SessionLineage{
				RootSessionID: "sess-root",
				PermissionPolicy: SessionPermissionPolicy{
					Tools: []string{"read", " "},
				},
			},
			want: "empty atom",
		},
		{
			name: "child missing root",
			id:   "sess-child",
			lineage: &SessionLineage{
				ParentSessionID: "sess-parent",
				SpawnDepth:      1,
			},
			want: "root session id is required",
		},
		{
			name: "root depth",
			id:   "sess-root",
			lineage: &SessionLineage{
				RootSessionID: "sess-root",
				SpawnDepth:    1,
			},
			want: "root session lineage depth",
		},
		{
			name: "root mismatch",
			id:   "sess-root",
			lineage: &SessionLineage{
				RootSessionID: "sess-other",
			},
			want: "root session lineage root",
		},
		{
			name: "root autostop",
			id:   "sess-root",
			lineage: &SessionLineage{
				RootSessionID:    "sess-root",
				AutoStopOnParent: true,
			},
			want: "cannot auto-stop",
		},
		{
			name: "child zero depth",
			id:   "sess-child",
			lineage: &SessionLineage{
				ParentSessionID: "sess-parent",
				RootSessionID:   "sess-root",
			},
			want: "depth must be greater",
		},
		{
			name: "child parent self",
			id:   "sess-child",
			lineage: &SessionLineage{
				ParentSessionID: "sess-child",
				RootSessionID:   "sess-root",
				SpawnDepth:      1,
			},
			want: "parent cannot be the session itself",
		},
		{
			name: "child root self",
			id:   "sess-child",
			lineage: &SessionLineage{
				ParentSessionID: "sess-parent",
				RootSessionID:   "sess-child",
				SpawnDepth:      1,
			},
			want: "root cannot be the session itself",
		},
		{
			name: "negative max depth",
			id:   "sess-root",
			lineage: &SessionLineage{
				RootSessionID: "sess-root",
				SpawnBudget:   SessionSpawnBudget{MaxDepth: -1},
			},
			want: "max_depth",
		},
		{
			name: "negative ttl seconds",
			id:   "sess-root",
			lineage: &SessionLineage{
				RootSessionID: "sess-root",
				SpawnBudget:   SessionSpawnBudget{TTLSeconds: -1},
			},
			want: "ttl_seconds",
		},
		{
			name: "negative active workspace cap",
			id:   "sess-root",
			lineage: &SessionLineage{
				RootSessionID: "sess-root",
				SpawnBudget:   SessionSpawnBudget{MaxActivePerWorkspace: -1},
			},
			want: "max_active_per_workspace",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSessionLineage(tt.id, tt.lineage)
			if err == nil {
				t.Fatalf("ValidateSessionLineage(%s) error = nil, want failure", tt.name)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("ValidateSessionLineage(%s) error = %v, want substring %q", tt.name, err, tt.want)
			}
		})
	}
}

func TestCloneSessionLineageReturnsDeepCopy(t *testing.T) {
	t.Parallel()

	ttl := time.Date(2026, 4, 26, 12, 0, 0, 0, time.FixedZone("UTC-3", -3*60*60))
	original := &SessionLineage{
		ParentSessionID: "sess-parent",
		RootSessionID:   "sess-root",
		SpawnDepth:      2,
		SpawnRole:       "worker",
		TTLExpiresAt:    &ttl,
		PermissionPolicy: SessionPermissionPolicy{
			Tools: []string{"write", "read"},
		},
	}

	cloned := CloneSessionLineage(original)
	if cloned == nil {
		t.Fatal("CloneSessionLineage() = nil, want copy")
	}
	if cloned == original {
		t.Fatal("CloneSessionLineage() returned original pointer")
	}
	if cloned.TTLExpiresAt == original.TTLExpiresAt {
		t.Fatal("CloneSessionLineage() reused TTL pointer")
	}
	if !cloned.TTLExpiresAt.Equal(ttl.UTC()) {
		t.Fatalf("cloned TTL = %s, want %s", cloned.TTLExpiresAt, ttl.UTC())
	}
	if cloned.PermissionPolicy.Tools[0] != "read" || cloned.PermissionPolicy.Tools[1] != "write" {
		t.Fatalf("cloned tools = %#v, want sorted policy atoms", cloned.PermissionPolicy.Tools)
	}

	original.PermissionPolicy.Tools[0] = "mutated"
	if cloned.PermissionPolicy.Tools[0] != "read" {
		t.Fatalf("cloned tools changed after original mutation: %#v", cloned.PermissionPolicy.Tools)
	}
}

func TestSessionLineageBudgetAndPolicyJSONRoundTrip(t *testing.T) {
	t.Parallel()

	budget := SessionSpawnBudget{
		MaxChildren:           2,
		MaxDepth:              3,
		TTLSeconds:            3600,
		MaxActivePerWorkspace: 4,
	}
	rawBudget, err := EncodeSessionSpawnBudget(budget)
	if err != nil {
		t.Fatalf("EncodeSessionSpawnBudget() error = %v", err)
	}
	decodedBudget, err := DecodeSessionSpawnBudget(rawBudget)
	if err != nil {
		t.Fatalf("DecodeSessionSpawnBudget() error = %v", err)
	}
	if decodedBudget != budget {
		t.Fatalf("decoded budget = %#v, want %#v", decodedBudget, budget)
	}
	emptyBudget, err := DecodeSessionSpawnBudget("  ")
	if err != nil {
		t.Fatalf("DecodeSessionSpawnBudget(empty) error = %v", err)
	}
	if emptyBudget != (SessionSpawnBudget{}) {
		t.Fatalf("DecodeSessionSpawnBudget(empty) = %#v, want zero budget", emptyBudget)
	}

	policy := SessionPermissionPolicy{
		Tools:               []string{"write", " read ", "read"},
		Skills:              []string{"go"},
		MCPServers:          []string{"memory"},
		WorkspacePaths:      []string{"/repo"},
		NetworkChannels:     []string{"coord"},
		EnvironmentProfiles: []string{"local"},
	}
	rawPolicy, err := EncodeSessionPermissionPolicy(policy)
	if err != nil {
		t.Fatalf("EncodeSessionPermissionPolicy() error = %v", err)
	}
	decodedPolicy, err := DecodeSessionPermissionPolicy(rawPolicy)
	if err != nil {
		t.Fatalf("DecodeSessionPermissionPolicy() error = %v", err)
	}
	wantPolicy := SessionPermissionPolicy{
		Tools:               []string{"read", "write"},
		Skills:              []string{"go"},
		MCPServers:          []string{"memory"},
		WorkspacePaths:      []string{"/repo"},
		NetworkChannels:     []string{"coord"},
		EnvironmentProfiles: []string{"local"},
	}
	if !reflect.DeepEqual(decodedPolicy, wantPolicy) {
		t.Fatalf("decoded policy = %#v, want %#v", decodedPolicy, wantPolicy)
	}
	emptyPolicy, err := DecodeSessionPermissionPolicy("")
	if err != nil {
		t.Fatalf("DecodeSessionPermissionPolicy(empty) error = %v", err)
	}
	if !reflect.DeepEqual(emptyPolicy, NormalizeSessionPermissionPolicy(SessionPermissionPolicy{})) {
		t.Fatalf("DecodeSessionPermissionPolicy(empty) = %#v, want normalized empty policy", emptyPolicy)
	}
}

func TestSessionLineageJSONRejectsMalformedValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		run  func() error
		want string
	}{
		{
			name: "encode negative budget",
			run: func() error {
				_, err := EncodeSessionSpawnBudget(SessionSpawnBudget{MaxChildren: -1})
				return err
			},
			want: "max_children",
		},
		{
			name: "decode malformed budget",
			run: func() error {
				_, err := DecodeSessionSpawnBudget("{")
				return err
			},
			want: "parse session spawn budget",
		},
		{
			name: "decode negative budget",
			run: func() error {
				_, err := DecodeSessionSpawnBudget(`{"ttl_seconds":-1}`)
				return err
			},
			want: "ttl_seconds",
		},
		{
			name: "encode empty policy atom",
			run: func() error {
				_, err := EncodeSessionPermissionPolicy(SessionPermissionPolicy{Skills: []string{"go", " "}})
				return err
			},
			want: "empty atom",
		},
		{
			name: "decode malformed policy",
			run: func() error {
				_, err := DecodeSessionPermissionPolicy("{")
				return err
			},
			want: "parse session permission policy",
		},
		{
			name: "decode empty policy atom",
			run: func() error {
				_, err := DecodeSessionPermissionPolicy(`{"network_channels":["coord"," "]}`)
				return err
			},
			want: "empty atom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run()
			if err == nil {
				t.Fatalf("%s error = nil, want failure", tt.name)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("%s error = %v, want substring %q", tt.name, err, tt.want)
			}
		})
	}
}
