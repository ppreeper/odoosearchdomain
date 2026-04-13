package odoosearchdomain

import (
	"reflect"
	"testing"
)

// searchDomainPatterns contains various test cases for the ParseDomain function.
var searchDomainPatterns = []struct {
	domain   string
	args     []any
	err      error
	expected []any
}{
	{"", []any{}, nil, []any{}},
	{"[]", []any{}, nil, []any{}},
	{"[()]", []any{}, nil, []any{}},
	{"('')", []any{}, nil, []any{}},
	{"('','')", []any{}, nil, []any{}},
	{"('a','=')", []any{}, nil, []any{}},
	{"('name')", []any{}, nil, []any{}},
	{"('name','=')", []any{}, nil, []any{}},

	{"[('name','=','My Name')]", []any{[]any{"name", "=", "My Name"}}, nil, []any{}},
	{"[('name','like','My Name')]", []any{[]any{"name", "like", "My Name"}}, nil, []any{}},
	{"[('name','ilike','My Name')]", []any{[]any{"name", "ilike", "My Name"}}, nil, []any{}},
	{"[('name','=','My Name'),('ref','=',12345)]", []any{[]any{"name", "=", "My Name"}, []any{"ref", "=", 12345}}, nil, []any{}},
	{"[('name','=','My Name'),('amount','=',123.45)]", []any{[]any{"name", "=", "My Name"}, []any{"amount", "=", 123.45}}, nil, []any{}},

	{"[('name', '=', 'John'), '|', ('is_company', '=', True), ('customer', '=', True)]", []any{[]any{"name", "=", "John"}, "|", []any{"is_company", "=", true}, []any{"customer", "=", true}}, nil, []any{}},
	{"[('name', 'like', 'John'),('ref', 'not like', 12345),('value','=',123.45),('customer','=',True)]", []any{[]any{"name", "like", "John"}, []any{"ref", "not like", 12345}, []any{"value", "=", 123.45}, []any{"customer", "=", true}}, nil, []any{}},
	{"[('name', 'like', 'John'),('ref', 'not like', 12345),('value','=',123.45),'&', ('is_company', '=', True),('customer','=',True)]", []any{[]any{"name", "like", "John"}, []any{"ref", "not like", 12345}, []any{"value", "=", 123.45}, "&", []any{"is_company", "=", true}, []any{"customer", "=", true}}, nil, []any{}},
	{"[('name', 'like', 'John'),('ref', 'not like', 12345),('value','=',123.45),'|', ('is_company', '=', True),('customer','=',True)]", []any{[]any{"name", "like", "John"}, []any{"ref", "not like", 12345}, []any{"value", "=", 123.45}, "|", []any{"is_company", "=", true}, []any{"customer", "=", true}}, nil, []any{}},
	{"[('name', '=', 'ABC'), '|', ('phone','ilike','7620'), ('mobile', 'ilike', '7620')]", []any{[]any{"name", "=", "ABC"}, "|", []any{"phone", "ilike", "7620"}, []any{"mobile", "ilike", "7620"}}, nil, []any{}},

	{"[('birthday.month_number', 'in', [10,11,12])]", []any{[]any{"birthday.month_number", "in", []any{10, 11, 12}}}, nil, []any{}},
	{"[('birthday.month', 'in', ['April','May','June'])]", []any{[]any{"birthday.month", "in", []any{"April", "May", "June"}}}, nil, []any{}},

	{"[('invoice_status', '=', 'to invoice'), ('order_line', 'any', [('product_id.qty_available', '<=', 0)])]", []any{[]any{"invoice_status", "=", "to invoice"}, []any{"order_line", "any", []any{[]any{"product_id.qty_available", "<=", 0}}}}, nil, []any{}},
	{"[('invoice_status', '=', 'to invoice'), ('order_line', 'any', [ ('product_id.qty_available', '<=', 0) , ('name','ilike','stud')])]", []any{[]any{"invoice_status", "=", "to invoice"}, []any{"order_line", "any", []any{[]any{"product_id.qty_available", "<=", 0}, []any{"name", "ilike", "stud"}}}}, nil, []any{}},

	{"[('name','lik','My Name')]", []any{}, ErrSyntax, []any{}},
	{"[('name', '=', 'ABC'), '|', ('phone','ilike','7620')]", []any{}, ErrNotEnoughAndOrTerms, []any{}},
	{"[('name', '=', 'ABC'), '!', ('phone','ilike','7620')]", []any{[]any{"name", "=", "ABC"}, "!", []any{"phone", "ilike", "7620"}}, nil, []any{}},
	{"[('name', '=', 'ABC'), '|', ('phone','ilike','7620'),'|']", []any{}, ErrNotEnoughAndOrTerms, []any{}},

	{"[('name', '=', 'ABC'), '!', ('phone','ilike','7620')]", []any{[]any{"name", "=", "ABC"}, "!", []any{"phone", "ilike", "7620"}}, nil, []any{}},
	{"[('name', '=', 'ABC'), '!']", []any{}, ErrNotEnoughNotTerms, []any{}},

	// Double-quoted strings
	{`[("name","=","My Name")]`, []any{[]any{"name", "=", "My Name"}}, nil, []any{}},
	{`[("name","ilike","John"),("ref","=",42)]`, []any{[]any{"name", "ilike", "John"}, []any{"ref", "=", 42}}, nil, []any{}},

	// Negative numbers
	{"[('qty','<=',-5)]", []any{[]any{"qty", "<=", -5}}, nil, []any{}},
	{"[('amount','=',-123.45)]", []any{[]any{"amount", "=", -123.45}}, nil, []any{}},

	// =like and =ilike operators
	{"[('name','=like','%abc%')]", []any{[]any{"name", "=like", "%abc%"}}, nil, []any{}},
	{"[('name','=ilike','%abc%')]", []any{[]any{"name", "=ilike", "%abc%"}}, nil, []any{}},

	// =? operator
	{"[('partner_id','=?',False)]", []any{[]any{"partner_id", "=?", false}}, nil, []any{}},

	// None/nil value
	{"[('parent_id','=',None)]", []any{[]any{"parent_id", "=", nil}}, nil, []any{}},

	// child_of and parent_of
	{"[('category_id','child_of',[5])]", []any{[]any{"category_id", "child_of", []any{5}}}, nil, []any{}},
	{"[('category_id','parent_of',3)]", []any{[]any{"category_id", "parent_of", 3}}, nil, []any{}},

	// Deeply nested prefix operators: & & t1 t2 t3
	{"['&', '&', ('a','=','1'), ('b','=','2'), ('c','=','3')]", []any{"&", "&", []any{"a", "=", "1"}, []any{"b", "=", "2"}, []any{"c", "=", "3"}}, nil, []any{}},

	// NOT combined with OR: ! | t1 t2
	{"['!', '|', ('a','=','1'), ('b','=','2')]", []any{"!", "|", []any{"a", "=", "1"}, []any{"b", "=", "2"}}, nil, []any{}},

	// Mixed quotes
	{`[('name','=',"O'Brien")]`, []any{[]any{"name", "=", "O'Brien"}}, nil, []any{}},

	// Empty list value
	{"[('ids','in',[])]", []any{[]any{"ids", "in", []any{}}}, nil, []any{}},

	// Value of 0
	{"[('qty','=',0)]", []any{[]any{"qty", "=", 0}}, nil, []any{}},
}

// TestSearchDomain tests the ParseDomain function with various search domain patterns.
func TestSearchDomain(t *testing.T) {
	for i, pattern := range searchDomainPatterns {
		args, err := ParseDomain(pattern.domain)
		if !reflect.DeepEqual(pattern.args, args) {
			t.Errorf("\n[%d]: domain: %s\nexpected reflect\nargs: %v\n got: %v", i, pattern.domain, pattern.args, args)
		}
		if err != pattern.err {
			t.Errorf("\n[%d]: domain: %v\nexpected error: %v\ngot %v", i, pattern.domain, pattern.err, err)
		}
	}
}

// --- Benchmarks ---

func BenchmarkParseDomain_Empty(b *testing.B) {
	for b.Loop() {
		ParseDomain("")
	}
}

func BenchmarkParseDomain_SingleTerm(b *testing.B) {
	const domain = "[('name','=','ABC')]"
	for b.Loop() {
		ParseDomain(domain)
	}
}

func BenchmarkParseDomain_ThreeTermsWithOr(b *testing.B) {
	const domain = "[('name', '=', 'ABC'), '|', ('phone','ilike','7620'), ('mobile', 'ilike', '7620')]"
	for b.Loop() {
		ParseDomain(domain)
	}
}

func BenchmarkParseDomain_SixTermsMixed(b *testing.B) {
	const domain = "[('name', 'like', 'John'),('ref', 'not like', 12345),('value','=',123.45),'|', ('is_company', '=', True),('customer','=',True)]"
	for b.Loop() {
		ParseDomain(domain)
	}
}

func BenchmarkParseDomain_NestedDomain(b *testing.B) {
	const domain = "[('invoice_status', '=', 'to invoice'), ('order_line', 'any', [ ('product_id.qty_available', '<=', 0) , ('name','ilike','stud')])]"
	for b.Loop() {
		ParseDomain(domain)
	}
}

func BenchmarkParseDomain_IntList(b *testing.B) {
	const domain = "[('birthday.month_number', 'in', [10,11,12])]"
	for b.Loop() {
		ParseDomain(domain)
	}
}

func BenchmarkParseDomain_StringList(b *testing.B) {
	const domain = "[('birthday.month', 'in', ['April','May','June'])]"
	for b.Loop() {
		ParseDomain(domain)
	}
}

func BenchmarkParseDomain_DeepPrefixNesting(b *testing.B) {
	const domain = "['&', '&', ('a','=','1'), ('b','=','2'), ('c','=','3')]"
	for b.Loop() {
		ParseDomain(domain)
	}
}

func BenchmarkParseDomain_NegativeNumbers(b *testing.B) {
	const domain = "[('qty','<=',-5),('amount','=',-123.45)]"
	for b.Loop() {
		ParseDomain(domain)
	}
}

func BenchmarkParseDomain_InvalidOperator(b *testing.B) {
	const domain = "[('name','lik','My Name')]"
	for b.Loop() {
		ParseDomain(domain)
	}
}

func BenchmarkParseDomain_ValidationError(b *testing.B) {
	const domain = "[('name', '=', 'ABC'), '|', ('phone','ilike','7620')]"
	for b.Loop() {
		ParseDomain(domain)
	}
}

func BenchmarkLexer(b *testing.B) {
	const domain = "[('name', 'like', 'John'),('ref', 'not like', 12345),('value','=',123.45),'|', ('is_company', '=', True),('customer','=',True)]"
	for b.Loop() {
		newLexer(domain).tokenize()
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
		ValidateDomain(terms)
	}
}
