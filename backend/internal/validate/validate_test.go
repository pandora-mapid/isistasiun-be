package validate

import "testing"

type sample struct {
	Name     string `json:"name" validate:"required"`
	Category string `json:"category" validate:"required,oneof=a b c"`
	Count    int    `json:"count" validate:"min=10000"`
}

func TestStructPassesWhenValid(t *testing.T) {
	if msg := Struct(&sample{Name: "x", Category: "b", Count: 10000}); msg != "" {
		t.Fatalf("expected no message, got %q", msg)
	}
}

func TestStructReportsJSONFieldName(t *testing.T) {
	msg := Struct(&sample{Category: "b", Count: 10000})
	if msg != "name is required" {
		t.Fatalf("got %q, want %q", msg, "name is required")
	}
}

func TestStructReportsOneofOptions(t *testing.T) {
	msg := Struct(&sample{Name: "x", Category: "z", Count: 10000})
	if msg != "category must be one of: a, b, c" {
		t.Fatalf("got %q", msg)
	}
}

func TestStructReportsMin(t *testing.T) {
	msg := Struct(&sample{Name: "x", Category: "a", Count: 500})
	if msg != "count must be at least 10000" {
		t.Fatalf("got %q", msg)
	}
}
