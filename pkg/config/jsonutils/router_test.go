// LLM usage: this test file was created with deepseek-v4-pro and modified manually.
package jsonutils

import (
	"encoding/json"
	"strings"
	"testing"
)

type testTypeA struct {
	Name string `json:"name"`
}

type testTypeB struct {
	ID int `json:"id"`
}

func TestJSONRouter_Unmarshal_HappyPath(t *testing.T) {
	mapping := JSONRouterMapping{
		"A": func() any { return &testTypeA{} },
		"B": func() any { return &testTypeB{} },
	}
	router := NewJSONRouter(mapping, "type")

	msg := json.RawMessage(`{"type":"A","name":"hello"}`)
	obj, value, err := router.Unmarshal(msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if value != "A" {
		t.Fatalf("expected value %q, got %q", "A", value)
	}
	a, ok := obj.(*testTypeA)
	if !ok {
		t.Fatalf("expected *testTypeA, got %T", obj)
	}
	if a.Name != "hello" {
		t.Fatalf("expected Name %q, got %q", "hello", a.Name)
	}
}

func TestJSONRouter_Unmarshal_MissingKey(t *testing.T) {
	mapping := JSONRouterMapping{
		"A": func() any { return &testTypeA{} },
	}
	router := NewJSONRouter(mapping, "type")

	msg := json.RawMessage(`{"name":"hello"}`)
	_, _, err := router.Unmarshal(msg)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "not in JSON context") {
		t.Fatalf("expected error to mention missing key, got: %v", err)
	}
}

func TestJSONRouter_Unmarshal_UnknownValue(t *testing.T) {
	mapping := JSONRouterMapping{
		"A": func() any { return &testTypeA{} },
	}
	router := NewJSONRouter(mapping, "type")

	msg := json.RawMessage(`{"type":"C"}`)
	_, _, err := router.Unmarshal(msg)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "unknown mapping string") {
		t.Fatalf("expected error to mention unknown mapping string, got: %v", err)
	}
}

func TestJSONRouter_Unmarshal_StructFieldsMismatch(t *testing.T) {
	mapping := JSONRouterMapping{
		"B": func() any { return &testTypeB{} },
	}
	router := NewJSONRouter(mapping, "type")

	// "id" is a string, but testTypeB.ID is int — should cause a type error.
	msg := json.RawMessage(`{"type":"B","id":"not-a-number"}`)
	_, _, err := router.Unmarshal(msg)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "cannot unmarshal") {
		t.Fatalf("expected a type-unmarshal error, got: %v", err)
	}
}
