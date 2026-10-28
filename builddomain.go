package odoosearchdomain

import "fmt"

// DomainList normalizes variadic domain arguments into a single []any.
// A single []any argument is returned as-is; multiple arguments are returned
// as a slice in call order. Unlike the elements themselves, no argument is
// ever discarded.
func DomainList(domains ...any) []any {
	if len(domains) == 0 {
		return []any{}
	}
	if len(domains) == 1 {
		if values, ok := domains[0].([]any); ok {
			return values
		}
	}
	return domains
}

// DomainString normalizes variadic string arguments into a []string.
// A single empty string yields an empty slice.
func DomainString(domains ...string) []string {
	if len(domains) == 0 || (len(domains) == 1 && domains[0] == "") {
		return []string{}
	}
	return domains
}

// Domain is a programmatically built search domain. Its elements are Term
// values, bare connector strings ("&", "|", "!"), or raw values appended by
// the caller. Use ToList to serialize it to the flat []any wire format.
type Domain []any

// NewDomain returns a pointer to a new, empty Domain. The builder methods use
// pointer receivers so they can append in place.
func NewDomain() *Domain {
	return &Domain{}
}

// ToList serializes the domain into the flat []any wire format: each Term
// becomes a []any tuple, and every other element (connector strings and raw
// values alike) is passed through unchanged — nothing is dropped.
func (dom *Domain) ToList() []any {
	list := make([]any, 0, len(*dom))
	for _, val := range *dom {
		if term, ok := val.(Term); ok {
			list = append(list, []any(term))
			continue
		}
		list = append(list, val)
	}
	return list
}

// AddTerm appends a new (field, operator, value) term to the domain.
func (dom *Domain) AddTerm(field, operator string, value any) *Domain {
	return dom.Add(NewTerm(field, operator, value))
}

// Add appends a pre-built term to the domain. A nil term is ignored.
func (dom *Domain) Add(term *Term) *Domain {
	if term == nil {
		return dom
	}
	*dom = append(*dom, *term)
	return dom
}

// AddConnector appends a bare logical connector to the domain.
// The operator must be one of "&", "|", or "!".
func (dom *Domain) AddConnector(operator string) (*Domain, error) {
	switch operator {
	case "&", "|", "!":
		*dom = append(*dom, operator)
		return dom, nil
	default:
		return dom, fmt.Errorf("invalid connector %q: must be \"&\", \"|\", or \"!\"", operator)
	}
}

// And appends the "&" connector followed by the two operand terms.
// In prefix notation the connector must precede its operands.
func (dom *Domain) And(term1, term2 *Term) *Domain {
	return dom.combinedTerms("&", term1, term2)
}

// Or appends the "|" connector followed by the two operand terms.
// In prefix notation the connector must precede its operands.
func (dom *Domain) Or(term1, term2 *Term) *Domain {
	return dom.combinedTerms("|", term1, term2)
}

// Not appends the "!" connector followed by its operand term.
// In prefix notation the connector must precede its operand.
func (dom *Domain) Not(term *Term) *Domain {
	return dom.combinedTerms("!", term)
}

// combinedTerms appends a connector followed by the given terms.
// Nil terms are skipped.
func (dom *Domain) combinedTerms(operator string, terms ...*Term) *Domain {
	*dom = append(*dom, operator)
	for _, term := range terms {
		if term == nil {
			continue
		}
		*dom = append(*dom, *term)
	}
	return dom
}

// Term is a single domain criterion: (field, operator, value).
type Term []any

// NewTerm creates a new term for the given field, operator, and value.
func NewTerm(field, operator string, value any) *Term {
	t := Term{field, operator, value}
	return &t
}
