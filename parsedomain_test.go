package odoosearchdomain

import (
	"reflect"
	"testing"
)

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
	{"[('name', '=', 'ABC'), '|', ('phone','ilike','7620')]", []any{[]any{"name", "=", "ABC"}, "|", []any{"phone", "ilike", "7620"}}, nil, []any{}},
	{"[('name', '=', 'ABC'), '!', ('phone','ilike','7620')]", []any{[]any{"name", "=", "ABC"}, "!", []any{"phone", "ilike", "7620"}}, nil, []any{}},
	{"[('name', '=', 'ABC'), '|', ('phone','ilike','7620'),'|']", []any{[]any{"name", "=", "ABC"}, "|", []any{"phone", "ilike", "7620"}, "|"}, nil, []any{}},

	{"[('name', '=', 'ABC'), '!', ('phone','ilike','7620')]", []any{[]any{"name", "=", "ABC"}, "!", []any{"phone", "ilike", "7620"}}, nil, []any{}},
	{"[('name', '=', 'ABC'), '!']", []any{[]any{"name", "=", "ABC"}, "!"}, nil, []any{}},
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

func TestValidateDomain(t *testing.T) {
	for i, pattern := range searchDomainPatterns {
		_, err := ParseDomain(pattern.domain)
		if err != nil && pattern.err == nil {
			t.Errorf("\n[%d]: expected no error for domain: %v\ngot %v", i, pattern.domain, err)
		}
		if err == nil && pattern.err != nil {
			t.Errorf("\n[%d]: expected error: %v\ndomain: %v\ngot no error", i, pattern.err, pattern.domain)
		}
	}
}
