package odoosearchdomain

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

// searchDomainPatterns contains various test cases for the ParseDomain function.
var searchDomainPatterns = []struct {
	domain string
	args   []any
	err    error
}{
	{"", []any{}, nil},
	{"[]", []any{}, nil},
	{"[()]", []any{}, nil},
	{"('')", []any{}, nil},
	{"('','')", []any{}, nil},
	{"('a','=')", []any{}, nil},
	{"('name')", []any{}, nil},
	{"('name','=')", []any{}, nil},

	{"[('name','=','My Name')]", []any{[]any{"name", "=", "My Name"}}, nil},
	{"[('name','like','My Name')]", []any{[]any{"name", "like", "My Name"}}, nil},
	{"[('name','ilike','My Name')]", []any{[]any{"name", "ilike", "My Name"}}, nil},
	{"[('name','=','My Name'),('ref','=',12345)]", []any{[]any{"name", "=", "My Name"}, []any{"ref", "=", 12345}}, nil},
	{"[('name','=','My Name'),('amount','=',123.45)]", []any{[]any{"name", "=", "My Name"}, []any{"amount", "=", 123.45}}, nil},

	{"[('name', '=', 'John'), '|', ('is_company', '=', True), ('customer', '=', True)]", []any{[]any{"name", "=", "John"}, "|", []any{"is_company", "=", true}, []any{"customer", "=", true}}, nil},
	{"[('name', 'like', 'John'),('ref', 'not like', 12345),('value','=',123.45),('customer','=',True)]", []any{[]any{"name", "like", "John"}, []any{"ref", "not like", 12345}, []any{"value", "=", 123.45}, []any{"customer", "=", true}}, nil},
	{"[('name', 'like', 'John'),('ref', 'not like', 12345),('value','=',123.45),'&', ('is_company', '=', True),('customer','=',True)]", []any{[]any{"name", "like", "John"}, []any{"ref", "not like", 12345}, []any{"value", "=", 123.45}, "&", []any{"is_company", "=", true}, []any{"customer", "=", true}}, nil},
	{"[('name', 'like', 'John'),('ref', 'not like', 12345),('value','=',123.45),'|', ('is_company', '=', True),('customer','=',True)]", []any{[]any{"name", "like", "John"}, []any{"ref", "not like", 12345}, []any{"value", "=", 123.45}, "|", []any{"is_company", "=", true}, []any{"customer", "=", true}}, nil},
	{"[('name', '=', 'ABC'), '|', ('phone','ilike','7620'), ('mobile', 'ilike', '7620')]", []any{[]any{"name", "=", "ABC"}, "|", []any{"phone", "ilike", "7620"}, []any{"mobile", "ilike", "7620"}}, nil},

	{"[('birthday.month_number', 'in', [10,11,12])]", []any{[]any{"birthday.month_number", "in", []any{10, 11, 12}}}, nil},
	{"[('birthday.month', 'in', ['April','May','June'])]", []any{[]any{"birthday.month", "in", []any{"April", "May", "June"}}}, nil},

	{"[('invoice_status', '=', 'to invoice'), ('order_line', 'any', [('product_id.qty_available', '<=', 0)])]", []any{[]any{"invoice_status", "=", "to invoice"}, []any{"order_line", "any", []any{[]any{"product_id.qty_available", "<=", 0}}}}, nil},
	{"[('invoice_status', '=', 'to invoice'), ('order_line', 'any', [ ('product_id.qty_available', '<=', 0) , ('name','ilike','stud')])]", []any{[]any{"invoice_status", "=", "to invoice"}, []any{"order_line", "any", []any{[]any{"product_id.qty_available", "<=", 0}, []any{"name", "ilike", "stud"}}}}, nil},

	{"[('name','lik','My Name')]", []any{}, ErrSyntax},
	{"[('name', '=', 'ABC'), '|', ('phone','ilike','7620')]", []any{}, ErrNotEnoughAndOrTerms},
	{"[('name', '=', 'ABC'), '!', ('phone','ilike','7620')]", []any{[]any{"name", "=", "ABC"}, "!", []any{"phone", "ilike", "7620"}}, nil},
	{"[('name', '=', 'ABC'), '|', ('phone','ilike','7620'),'|']", []any{}, ErrNotEnoughAndOrTerms},

	{"[('name', '=', 'ABC'), '!']", []any{}, ErrNotEnoughNotTerms},

	// Double-quoted strings
	{`[("name","=","My Name")]`, []any{[]any{"name", "=", "My Name"}}, nil},
	{`[("name","ilike","John"),("ref","=",42)]`, []any{[]any{"name", "ilike", "John"}, []any{"ref", "=", 42}}, nil},

	// Negative numbers
	{"[('qty','<=',-5)]", []any{[]any{"qty", "<=", -5}}, nil},
	{"[('amount','=',-123.45)]", []any{[]any{"amount", "=", -123.45}}, nil},

	// Scientific notation (Python's repr of large/small floats)
	{"[('amount','=',1e+21)]", []any{[]any{"amount", "=", 1e+21}}, nil},
	{"[('amount','=',1.5e-3)]", []any{[]any{"amount", "=", 1.5e-3}}, nil},
	{"[('amount','=',-2E5)]", []any{[]any{"amount", "=", -2e5}}, nil},

	// Python integers are unbounded; int overflow falls back to float64
	{"[('amount','=',99999999999999999999)]", []any{[]any{"amount", "=", 1e20}}, nil},

	// =like and =ilike operators
	{"[('name','=like','%abc%')]", []any{[]any{"name", "=like", "%abc%"}}, nil},
	{"[('name','=ilike','%abc%')]", []any{[]any{"name", "=ilike", "%abc%"}}, nil},

	// =? operator
	{"[('partner_id','=?',False)]", []any{[]any{"partner_id", "=?", false}}, nil},

	// None/nil value
	{"[('parent_id','=',None)]", []any{[]any{"parent_id", "=", nil}}, nil},

	// child_of and parent_of
	{"[('category_id','child_of',[5])]", []any{[]any{"category_id", "child_of", []any{5}}}, nil},
	{"[('category_id','parent_of',3)]", []any{[]any{"category_id", "parent_of", 3}}, nil},

	// Deeply nested prefix operators: & & t1 t2 t3
	{"['&', '&', ('a','=','1'), ('b','=','2'), ('c','=','3')]", []any{"&", "&", []any{"a", "=", "1"}, []any{"b", "=", "2"}, []any{"c", "=", "3"}}, nil},

	// NOT combined with OR: ! | t1 t2
	{"['!', '|', ('a','=','1'), ('b','=','2')]", []any{"!", "|", []any{"a", "=", "1"}, []any{"b", "=", "2"}}, nil},

	// Mixed quotes
	{`[('name','=',"O'Brien")]`, []any{[]any{"name", "=", "O'Brien"}}, nil},

	// Python escape sequences are decoded
	{"[('name','=','a\\nb')]", []any{[]any{"name", "=", "a\nb"}}, nil},
	{`[('name','=','Line1\nLine2')]`, []any{[]any{"name", "=", "Line1\nLine2"}}, nil},
	{`[('name','=','tab\there')]`, []any{[]any{"name", "=", "tab\there"}}, nil},
	{`[('name','=','\x41\x42')]`, []any{[]any{"name", "=", "AB"}}, nil},
	{`[('name','=','caf\u00e9')]`, []any{[]any{"name", "=", "café"}}, nil},
	{`[('name','=','\U0001F600')]`, []any{[]any{"name", "=", "😀"}}, nil},
	{`[('name','=','a\qb')]`, []any{[]any{"name", "=", `a\qb`}}, nil}, // unknown escape passes through
	{`[("name","=","say \"hi\"")]`, []any{[]any{"name", "=", `say "hi"`}}, nil},

	// Empty list value
	{"[('ids','in',[])]", []any{[]any{"ids", "in", []any{}}}, nil},

	// Value of 0
	{"[('qty','=',0)]", []any{[]any{"qty", "=", 0}}, nil},

	// Errors on bracketed input are reported, not swallowed
	{"[('name','=','abc)]", []any{}, ErrSyntax},            // unterminated string
	{`[("name","=","abc)]`, []any{}, ErrSyntax},            // unterminated double-quoted string
	{"[('a','=','1')] garbage", []any{}, ErrSyntax},        // trailing characters after ']'
	{"[('a','=','1'),]", []any{[]any{"a", "=", "1"}}, nil}, // trailing comma is allowed

	// Invalid Unicode escapes are rejected, not silently replaced
	{`[('a','=','\U00110000')]`, []any{}, ErrSyntax}, // out of range
	{`[('a','=','\uD800')]`, []any{}, ErrSyntax},     // surrogate

	// Nested domain arity is validated
	{"[('x','any',['!'])]", []any{}, ErrNotEnoughNotTerms},
	{"[('x','any',['&',('a','=',1)])]", []any{}, ErrNotEnoughAndOrTerms},
	{"[('x','any',['|',('a','=',1),('b','=',2)])]", []any{[]any{"x", "any", []any{"|", []any{"a", "=", 1}, []any{"b", "=", 2}}}}, nil},

	// A plain list value starting with a connector character is not a domain
	{"[('x','in',['&','a'])]", []any{[]any{"x", "in", []any{"&", "a"}}}, nil},
}

// TestSearchDomain tests the ParseDomain function with various search domain patterns.
func TestSearchDomain(t *testing.T) {
	for i, pattern := range searchDomainPatterns {
		args, err := ParseDomain(pattern.domain)
		if !reflect.DeepEqual(pattern.args, args) {
			t.Errorf("\n[%d]: domain: %s\nexpected reflect\nargs: %v\n got: %v", i, pattern.domain, pattern.args, args)
		}
		if !errors.Is(err, pattern.err) {
			t.Errorf("\n[%d]: domain: %v\nexpected error: %v\ngot %v", i, pattern.domain, pattern.err, err)
		}
	}
}

// TestParseDomain_NestedValidation verifies that errors inside a nested domain
// are reported with their real cause rather than masked by the list fallback.
func TestParseDomain_NestedValidation(t *testing.T) {
	_, err := ParseDomain("[('x','any',[('a','badop',1)])]")
	if !errors.Is(err, ErrSyntax) {
		t.Fatalf("expected ErrSyntax, got: %v", err)
	}
	if !strings.Contains(err.Error(), "badop") {
		t.Errorf("expected error to name the bad operator, got: %v", err)
	}
}

// TestParseDomain_InvalidEscape verifies that out-of-range code points are
// rejected instead of being silently replaced with U+FFFD.
func TestParseDomain_InvalidEscape(t *testing.T) {
	_, err := ParseDomain(`[('a','=','\U00110000')]`)
	if !errors.Is(err, ErrSyntax) {
		t.Fatalf("expected ErrSyntax for out-of-range code point, got: %v", err)
	}
	if strings.Contains(err.Error(), "\uFFFD") {
		t.Errorf("error message contains a replacement character: %q", err.Error())
	}
}

// FuzzParseDomain asserts the parser's core invariants: it must not panic,
// every error must be one of the sentinel errors, and anything it accepts
// must pass ValidateDomain.
func FuzzParseDomain(f *testing.F) {
	for _, seed := range []string{
		"", "[]", "[()]", "[('name','=','x')]",
		"[('a','in',[1,2])]", "['|',('a','=',1),('b','=',2)]",
		"[('a','any',[('b','=',1)])]", `[('a','=','\x41')]`,
		"[('a','any',['!'])]", "[('a','=','x')] garbage",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, s string) {
		res, err := ParseDomain(s)
		if err != nil {
			if res == nil {
				t.Fatalf("nil result with error %v (input %q)", err, s)
			}
			switch {
			case errors.Is(err, ErrSyntax),
				errors.Is(err, ErrNotEnoughAndOrTerms),
				errors.Is(err, ErrNotEnoughNotTerms):
			default:
				t.Fatalf("unexpected error type %v (input %q)", err, s)
			}
			return
		}
		if len(res) > 0 {
			if _, verr := ValidateDomain(res); verr != nil {
				t.Fatalf("accepted %q but validation failed: %v", s, verr)
			}
		}
	})
}

// TestParseDomain_NestingLimit verifies that deeply nested list values are
// rejected with ErrSyntax instead of exhausting the stack.
func TestParseDomain_NestingLimit(t *testing.T) {
	makeNested := func(depth int) string {
		return "[('field', 'in', " + strings.Repeat("[", depth) + "1" + strings.Repeat("]", depth) + ")]"
	}

	if _, err := ParseDomain(makeNested(maxNestingDepth - 1)); err != nil {
		t.Errorf("expected nesting depth %d to parse without error, got: %v", maxNestingDepth-1, err)
	}
	if _, err := ParseDomain(makeNested(maxNestingDepth + 1)); !errors.Is(err, ErrSyntax) {
		t.Errorf("expected ErrSyntax for nesting depth %d, got: %v", maxNestingDepth+1, err)
	}
}

// TestValidateDomain_NestingLimit verifies that deeply chained prefix
// operators are rejected with ErrSyntax instead of exhausting the stack.
func TestValidateDomain_NestingLimit(t *testing.T) {
	makeChain := func(n int) []any {
		terms := make([]any, 0, 2*n+1)
		for i := 0; i < n; i++ {
			terms = append(terms, "&")
		}
		for i := 0; i <= n; i++ {
			terms = append(terms, []any{"field", "=", i})
		}
		return terms
	}

	if _, err := ValidateDomain(makeChain(maxNestingDepth - 1)); err != nil {
		t.Errorf("expected operator chain of %d to validate, got: %v", maxNestingDepth-1, err)
	}
	if _, err := ValidateDomain(makeChain(maxNestingDepth + 1)); !errors.Is(err, ErrSyntax) {
		t.Errorf("expected ErrSyntax for operator chain of %d, got: %v", maxNestingDepth+1, err)
	}
}

// --- Benchmarks ---

func BenchmarkParseDomain_Empty(b *testing.B) {
	for b.Loop() {
		_, _ = ParseDomain("")
	}
}

func BenchmarkParseDomain_SingleTerm(b *testing.B) {
	const domain = "[('name','=','ABC')]"
	for b.Loop() {
		_, _ = ParseDomain(domain)
	}
}

func BenchmarkParseDomain_ThreeTermsWithOr(b *testing.B) {
	const domain = "[('name', '=', 'ABC'), '|', ('phone','ilike','7620'), ('mobile', 'ilike', '7620')]"
	for b.Loop() {
		_, _ = ParseDomain(domain)
	}
}

func BenchmarkParseDomain_SixTermsMixed(b *testing.B) {
	const domain = "[('name', 'like', 'John'),('ref', 'not like', 12345),('value','=',123.45),'|', ('is_company', '=', True),('customer','=',True)]"
	for b.Loop() {
		_, _ = ParseDomain(domain)
	}
}

func BenchmarkParseDomain_NestedDomain(b *testing.B) {
	const domain = "[('invoice_status', '=', 'to invoice'), ('order_line', 'any', [ ('product_id.qty_available', '<=', 0) , ('name','ilike','stud')])]"
	for b.Loop() {
		_, _ = ParseDomain(domain)
	}
}

func BenchmarkParseDomain_IntList(b *testing.B) {
	const domain = "[('birthday.month_number', 'in', [10,11,12])]"
	for b.Loop() {
		_, _ = ParseDomain(domain)
	}
}

func BenchmarkParseDomain_StringList(b *testing.B) {
	const domain = "[('birthday.month', 'in', ['April','May','June'])]"
	for b.Loop() {
		_, _ = ParseDomain(domain)
	}
}

func BenchmarkParseDomain_DeepPrefixNesting(b *testing.B) {
	const domain = "['&', '&', ('a','=','1'), ('b','=','2'), ('c','=','3')]"
	for b.Loop() {
		_, _ = ParseDomain(domain)
	}
}

func BenchmarkParseDomain_NegativeNumbers(b *testing.B) {
	const domain = "[('qty','<=',-5),('amount','=',-123.45)]"
	for b.Loop() {
		_, _ = ParseDomain(domain)
	}
}

func BenchmarkParseDomain_InvalidOperator(b *testing.B) {
	const domain = "[('name','lik','My Name')]"
	for b.Loop() {
		_, _ = ParseDomain(domain)
	}
}

func BenchmarkParseDomain_ValidationError(b *testing.B) {
	const domain = "[('name', '=', 'ABC'), '|', ('phone','ilike','7620')]"
	for b.Loop() {
		_, _ = ParseDomain(domain)
	}
}

func BenchmarkLexer(b *testing.B) {
	const domain = "[('name', 'like', 'John'),('ref', 'not like', 12345),('value','=',123.45),'|', ('is_company', '=', True),('customer','=',True)]"
	for b.Loop() {
		_, _ = newLexer(domain).tokenize()
	}
}

func BenchmarkValidateDomain(b *testing.B) {
	terms := []any{
		[]any{"name", "like", "John"},
		[]any{"ref", "not like", 12345},
		[]any{"value", "=", 123.45},
		"|",
		[]any{"is_company", "=", true},
		[]any{"customer", "=", true},
	}
	for b.Loop() {
		_, _ = ValidateDomain(terms)
	}
}
