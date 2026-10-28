package odoosearchdomain

import (
	"reflect"
	"testing"
)

func TestDomainList(t *testing.T) {
	cases := []struct {
		name     string
		input    []any
		expected []any
	}{
		{"no args", nil, []any{}},
		{"single slice", []any{[]any{"a", "=", 1}}, []any{"a", "=", 1}},
		{"single empty slice", []any{[]any{}}, []any{}},
		{"multiple scalars", []any{"a", "b"}, []any{"a", "b"}},
		{"slice plus extra is kept", []any{[]any{"a"}, "b"}, []any{[]any{"a"}, "b"}},
	}
	for _, tc := range cases {
		var got []any
		if tc.input == nil {
			got = DomainList()
		} else {
			got = DomainList(tc.input...)
		}
		if !reflect.DeepEqual(got, tc.expected) {
			t.Errorf("%s: expected %v, got %v", tc.name, tc.expected, got)
		}
	}
}

func TestDomainString(t *testing.T) {
	cases := []struct {
		name     string
		input    []string
		expected []string
	}{
		{"no args", nil, []string{}},
		{"single empty string", []string{""}, []string{}},
		{"single string", []string{"a"}, []string{"a"}},
		{"multiple strings", []string{"a", "b"}, []string{"a", "b"}},
	}
	for _, tc := range cases {
		var got []string
		if tc.input == nil {
			got = DomainString()
		} else {
			got = DomainString(tc.input...)
		}
		if !reflect.DeepEqual(got, tc.expected) {
			t.Errorf("%s: expected %v, got %v", tc.name, tc.expected, got)
		}
	}
}

func TestDomainBuilder(t *testing.T) {
	dom := NewDomain()
	dom.AddTerm("name", "=", "ABC")
	dom.Or(
		NewTerm("phone", "ilike", "7620"),
		NewTerm("mobile", "ilike", "7620"),
	)

	expected := []any{
		[]any{"name", "=", "ABC"},
		"|",
		[]any{"phone", "ilike", "7620"},
		[]any{"mobile", "ilike", "7620"},
	}
	if got := dom.ToList(); !reflect.DeepEqual(got, expected) {
		t.Errorf("expected %v, got %v", expected, got)
	}
}

func TestDomainAnd(t *testing.T) {
	dom := NewDomain()
	dom.And(NewTerm("a", "=", 1), NewTerm("b", "=", 2))

	expected := []any{"&", []any{"a", "=", 1}, []any{"b", "=", 2}}
	if got := dom.ToList(); !reflect.DeepEqual(got, expected) {
		t.Errorf("expected %v, got %v", expected, got)
	}
}

func TestDomainNot(t *testing.T) {
	dom := NewDomain()
	dom.Not(NewTerm("active", "=", false))

	expected := []any{"!", []any{"active", "=", false}}
	if got := dom.ToList(); !reflect.DeepEqual(got, expected) {
		t.Errorf("expected %v, got %v", expected, got)
	}
}

func TestDomainAddConnector(t *testing.T) {
	dom := NewDomain()
	if _, err := dom.AddConnector("!"); err != nil {
		t.Fatalf("expected valid connector, got error: %v", err)
	}
	if _, err := dom.AddConnector("bad"); err == nil {
		t.Error("expected error for invalid connector, got nil")
	}

	expected := []any{"!"}
	if got := dom.ToList(); !reflect.DeepEqual(got, expected) {
		t.Errorf("expected %v, got %v", expected, got)
	}
}

func TestDomainAddNil(t *testing.T) {
	dom := NewDomain()
	dom.Add(nil)
	dom.Not(nil)

	expected := []any{"!"}
	if got := dom.ToList(); !reflect.DeepEqual(got, expected) {
		t.Errorf("expected %v, got %v", expected, got)
	}
}

func TestToListPassthrough(t *testing.T) {
	dom := NewDomain()
	dom.AddTerm("name", "=", "ABC")
	// Raw elements appended directly (e.g. from ParseDomain output) pass through.
	*dom = append(*dom, "!", []any{"active", "=", true})

	expected := []any{
		[]any{"name", "=", "ABC"},
		"!",
		[]any{"active", "=", true},
	}
	if got := dom.ToList(); !reflect.DeepEqual(got, expected) {
		t.Errorf("expected %v, got %v", expected, got)
	}
}

func TestFields(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected []string
	}{
		{"empty", "", []string{}},
		{"simple", "a,b", []string{"a", "b"}},
		{"spaces trimmed", "a, b , c", []string{"a", "b", "c"}},
		{"empty entries dropped", "a,,b", []string{"a", "b"}},
		{"only commas", ",,", []string{}},
	}
	for _, tc := range cases {
		if got := Fields(tc.input); !reflect.DeepEqual(got, tc.expected) {
			t.Errorf("%s: expected %v, got %v", tc.name, tc.expected, got)
		}
	}
}

func TestRoundTrip(t *testing.T) {
	dom := NewDomain()
	dom.AddTerm("name", "=", "ABC")
	dom.Or(
		NewTerm("phone", "ilike", "7620"),
		NewTerm("mobile", "ilike", "7620"),
	)

	list := dom.ToList()
	parsed, err := ParseDomain(`[('name','=','ABC'), '|', ('phone','ilike','7620'), ('mobile','ilike','7620')]`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(list, parsed) {
		t.Errorf("builder output does not match parsed domain\nbuilt:  %v\nparsed: %v", list, parsed)
	}
}
