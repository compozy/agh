package workref

import "testing"

type constructorCase struct {
	name      string
	id        string
	value     string
	wantID    string
	wantValue string
}

var constructorCases = []constructorCase{
	{
		name:      "trims leading and trailing whitespace",
		id:        "  workspace-01\t",
		value:     "\n /tmp/demo \t",
		wantID:    "workspace-01",
		wantValue: "/tmp/demo",
	},
	{
		name:      "preserves interior whitespace",
		id:        "workspace 01",
		value:     "/tmp/demo folder",
		wantID:    "workspace 01",
		wantValue: "/tmp/demo folder",
	},
	{
		name:      "collapses whitespace only values to empty strings",
		id:        " \n\t ",
		value:     "\t ",
		wantID:    "",
		wantValue: "",
	},
}

func runConstructorCases[T comparable](
	t *testing.T,
	constructor func(string, string) T,
	want func(string, string) T,
) {
	t.Helper()

	for _, tc := range constructorCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := constructor(tc.id, tc.value)
			expected := want(tc.wantID, tc.wantValue)
			if got != expected {
				t.Fatalf("constructor(%q, %q) = %#v, want %#v", tc.id, tc.value, got, expected)
			}
		})
	}
}

func runConstructorSuite[T comparable](
	t *testing.T,
	name string,
	constructor func(string, string) T,
	want func(string, string) T,
) {
	t.Helper()

	t.Run(name, func(t *testing.T) {
		t.Parallel()
		runConstructorCases(t, constructor, want)
	})
}

func TestConstructors(t *testing.T) {
	t.Parallel()

	runConstructorSuite(t, "NewPath", NewPath, func(id string, value string) PathRef {
		return PathRef{
			WorkspaceID:   id,
			WorkspacePath: value,
		}
	})

	runConstructorSuite(t, "NewRoot", NewRoot, func(id string, value string) RootRef {
		return RootRef{
			WorkspaceID: id,
			Workspace:   value,
		}
	})
}
