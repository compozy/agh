package extension_test

import "testing"

func TestNonEmptyLines(t *testing.T) {
	t.Parallel()

	got := nonEmptyLines("\n first \n\nsecond\n  \n third  \n")
	want := []string{"first", "second", "third"}
	if len(got) != len(want) {
		t.Fatalf("len(nonEmptyLines()) = %d, want %d", len(got), len(want))
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("nonEmptyLines()[%d] = %q, want %q", index, got[index], want[index])
		}
	}
}

func TestContainsFragmentsInOrder(t *testing.T) {
	t.Parallel()

	if !containsFragmentsInOrder("alpha beta gamma", "alpha", "beta", "gamma") {
		t.Fatal("containsFragmentsInOrder() = false, want true for ordered fragments")
	}
	if containsFragmentsInOrder("alpha gamma beta", "alpha", "beta", "gamma") {
		t.Fatal("containsFragmentsInOrder() = true, want false for out-of-order fragments")
	}
}

func TestDecodeJSONLines(t *testing.T) {
	t.Parallel()

	type sample struct {
		Name string `json:"name"`
	}

	items, err := decodeJSONLines[sample]([]byte("{\"name\":\"alpha\"}\n\n{\"name\":\"beta\"}\n"))
	if err != nil {
		t.Fatalf("decodeJSONLines() error = %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("len(decodeJSONLines()) = %d, want 2", len(items))
	}
	if items[0].Name != "alpha" || items[1].Name != "beta" {
		t.Fatalf("decodeJSONLines() = %#v, want alpha/beta", items)
	}

	if _, err := decodeJSONLines[sample]([]byte("{not-json}\n")); err == nil {
		t.Fatal("decodeJSONLines(invalid) error = nil, want non-nil")
	}
}
