package config

import "testing"

func TestStringDefault(t *testing.T) {
	t.Setenv("ARTICLEFLOW_TEST_VALUE", "")

	value := String("ARTICLEFLOW_TEST_VALUE", "fallback")

	if value != "fallback" {
		t.Fatalf("expected fallback, got %q", value)
	}
}

func TestStringOverride(t *testing.T) {
	t.Setenv("ARTICLEFLOW_TEST_VALUE", "custom")

	value := String("ARTICLEFLOW_TEST_VALUE", "fallback")

	if value != "custom" {
		t.Fatalf("expected custom, got %q", value)
	}
}

func TestIntDefaultAndOverride(t *testing.T) {
	t.Setenv("ARTICLEFLOW_TEST_INT", "")
	if value := Int("ARTICLEFLOW_TEST_INT", 8080); value != 8080 {
		t.Fatalf("expected default 8080, got %d", value)
	}

	t.Setenv("ARTICLEFLOW_TEST_INT", "9090")
	if value := Int("ARTICLEFLOW_TEST_INT", 8080); value != 9090 {
		t.Fatalf("expected override 9090, got %d", value)
	}
}

