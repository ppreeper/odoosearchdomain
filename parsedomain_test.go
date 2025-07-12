package odoosearchdomain

import (
	"reflect"
	"testing"
)

var searchDomainPatterns = []struct {
	domain string
	args   []any
	err    error
}{
	{"", []any{}, ErrSyntax},
	{"[]", []any{}, ErrSyntax},
	{"[()]", []any{}, ErrSyntax},
	{"('')", []any{}, ErrSyntax},
	{"('','')", []any{}, ErrSyntax},
	{"('a','=')", []any{}, ErrSyntax},
	{"('name')", []any{}, ErrSyntax},
	{"('name','=')", []any{}, ErrSyntax},

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

	{"[('name','lik','My Name')]", []any{}, ErrInvalidComparator},
	{"[('name', '=', 'ABC'), '|', ('phone','ilike','7620')]", []any{}, ErrNotEnoughAndOrTerms},
	{"[('name', '=', 'ABC'), '|', ('phone','ilike','7620'),'|']", []any{}, ErrSyntax},

	{"[('name', '=', 'ABC'), '!', ('phone','ilike','7620')]", []any{[]any{"name", "=", "ABC"}, "!", []any{"phone", "ilike", "7620"}}, nil},
	{"[('name', '=', 'ABC'), '!']", []any{}, ErrSyntax},
}

func TestSearchDomain(t *testing.T) {
	for i, pattern := range searchDomainPatterns {
		args, err := ParseDomain(pattern.domain)
		if !reflect.DeepEqual(pattern.args, args) {
			t.Errorf("\n[%d]: expected reflect\nargs: %v\n got: %v", i, pattern.args, args)
		}
		if err != pattern.err {
			t.Errorf("\n[%d]: expected error: %v\ndomain: %v\ngot %v", i, pattern.err, pattern.domain, err)
		}
	}
}
