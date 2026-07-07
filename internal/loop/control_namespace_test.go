package loop

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/compozy/agh/internal/loop/dsl"
)

func TestCollectionItemsShouldResolveFiniteFanOutArrays(t *testing.T) {
	t.Parallel()

	t.Run("Should decode JSON string collections", func(t *testing.T) {
		t.Parallel()

		items, err := collectionItems(`[{"id":"A"},{"id":"B"}]`)
		if err != nil {
			t.Fatalf("collectionItems() error = %v", err)
		}
		if got, want := len(items), 2; got != want {
			t.Fatalf("items len = %d, want %d", got, want)
		}
		first, ok := items[0].(map[string]any)
		if !ok {
			t.Fatalf("first item = %#v, want object", items[0])
		}
		if got, want := first["id"], "A"; got != want {
			t.Fatalf("first id = %v, want %q", got, want)
		}
	})

	t.Run("Should decode raw JSON collections", func(t *testing.T) {
		t.Parallel()

		items, err := collectionItems(json.RawMessage(`[1,2,3]`))
		if err != nil {
			t.Fatalf("collectionItems() error = %v", err)
		}
		if got, want := len(items), 3; got != want {
			t.Fatalf("items len = %d, want %d", got, want)
		}
	})

	t.Run("Should copy typed slice collections", func(t *testing.T) {
		t.Parallel()

		items, err := collectionItems([]string{"A", "B"})
		if err != nil {
			t.Fatalf("collectionItems() error = %v", err)
		}
		if got, want := len(items), 2; got != want {
			t.Fatalf("items len = %d, want %d", got, want)
		}
		if got, want := items[1], "B"; got != want {
			t.Fatalf("second item = %v, want %q", got, want)
		}
	})

	t.Run("Should reject non array collections", func(t *testing.T) {
		t.Parallel()

		_, err := collectionItems(`{"id":"A"}`)
		if !errors.Is(err, ErrValidation) {
			t.Fatalf("collectionItems() error = %v, want ErrValidation", err)
		}
	})
}

func TestResolveFanOutCollectionShouldRenderTemplateCollections(t *testing.T) {
	t.Parallel()

	t.Run("Should render template collections", func(t *testing.T) {
		t.Parallel()

		node := dsl.Node{
			ID:         "fan",
			Collection: `{{ json .nodes.load.output.items }}`,
		}
		namespace := map[string]any{
			"nodes": map[string]any{
				"load": map[string]any{
					"output": map[string]any{
						"items": []any{"A", "B"},
					},
				},
			},
		}

		items, err := resolveFanOutCollection(&ResolvedDefinition{}, node, namespace)
		if err != nil {
			t.Fatalf("resolveFanOutCollection() error = %v", err)
		}
		if got, want := len(items), 2; got != want {
			t.Fatalf("items len = %d, want %d", got, want)
		}
		if got, want := items[0], "A"; got != want {
			t.Fatalf("first item = %v, want %q", got, want)
		}
	})
}

func TestOutputValueShouldPreserveExactJSONNumbers(t *testing.T) {
	t.Parallel()

	t.Run("Should preserve large JSON numbers as json.Number", func(t *testing.T) {
		t.Parallel()

		value := outputValue(`{"id":9007199254740993}`)
		object, ok := value.(map[string]any)
		if !ok {
			t.Fatalf("outputValue() = %#v, want object", value)
		}
		number, ok := object["id"].(json.Number)
		if !ok {
			t.Fatalf("outputValue().id = %#v, want json.Number", object["id"])
		}
		if got, want := number.String(), "9007199254740993"; got != want {
			t.Fatalf("outputValue().id = %q, want %q", got, want)
		}
	})
}
