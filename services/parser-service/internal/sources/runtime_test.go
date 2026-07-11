package sources

import (
	"testing"

	parserv1 "github.com/hanq/articleflow/contracts/parser/v1"
)

func TestRuntimeRegistryCanDisableAndEnableSource(t *testing.T) {
	registry := NewRuntimeRegistry([]parserv1.ParserSource{
		{Name: "habr", DisplayName: "Habr", Enabled: true, Searchable: true},
		{Name: "vc", DisplayName: "vc.ru", Enabled: true, Searchable: true},
	})

	updated, ok := registry.SetEnabled("vc", false)
	if !ok {
		t.Fatal("expected vc source to be found")
	}
	if updated.Enabled {
		t.Fatalf("expected vc disabled, got %#v", updated)
	}
	if registry.Enabled("vc") {
		t.Fatal("expected runtime gate to disable vc")
	}

	updated, ok = registry.SetEnabled("vc", true)
	if !ok {
		t.Fatal("expected vc source to be found")
	}
	if !updated.Enabled || !registry.Enabled("vc") {
		t.Fatalf("expected vc enabled again, got %#v", updated)
	}
}

func TestRuntimeRegistryListReturnsCopy(t *testing.T) {
	registry := NewRuntimeRegistry([]parserv1.ParserSource{
		{Name: "habr", DisplayName: "Habr", Enabled: true, Searchable: true},
	})

	sources := registry.List()
	sources[0].Enabled = false

	if !registry.Enabled("habr") {
		t.Fatal("mutating returned list must not mutate registry")
	}
}
