package odoosearchdomain

func DomainList(Domains ...any) []any {
	if len(Domains) == 0 {
		return []any{}
	}
	if len(Domains) == 1 {
		switch values := Domains[0].(type) {
		case []any:
			if len(values) == 0 {
				return []any{}
			} else {
				return values
			}
		}
	}
	return Domains[0].([]any)
}

func DomainString(Domains ...string) []string {
	if len(Domains) == 0 {
		return []string{}
	}
	if len(Domains) == 1 && len(Domains[0]) == 0 {
		return []string{}
	}
	return Domains
}

type Domain []any

func NewDomain() *Domain {
	return &Domain{}
}

func (dom *Domain) ToList() []any {
	Domain := []any{}
	for _, val := range *dom {
		switch valType := val.(type) {
		case Term:
			DomainTerm := []any{}
			for _, vv := range valType {
				DomainTerm = append(DomainTerm, vv)
			}
			Domain = append(Domain, DomainTerm)
		case string:
			Domain = append(Domain, val)
		}
	}
	return Domain
}

func (dom *Domain) AddTerm(field, operator string, value any) *Domain {
	*dom = append(*dom, *NewTerm(field, operator, value))
	return dom
}

func (dom *Domain) Add(term *Term) *Domain {
	*dom = append(*dom, term)
	return dom
}

func (dom *Domain) And(term1, term2 *Term) *Domain {
	return dom.combinedTerms("&", term1, term2)
}

func (dom *Domain) Or(term1, term2 *Term) *Domain {
	return dom.combinedTerms("|", term1, term2)
}

func (dom *Domain) Not(term *Term) *Domain {
	return dom.combinedTerms("!", term)
}

func (dom *Domain) combinedTerms(operator string, terms ...*Term) *Domain {
	*dom = append(*dom, operator)
	for _, term := range terms {
		*dom = append(*dom, term)
	}
	return dom
}

type Term []any

func NewTerm(field, operator string, value any) *Term {
	c := Term(newTuple(field, operator, value))
	return &c
}

func newTuple(values ...any) []any {
	t := make([]any, len(values))
	copy(t, values)
	return t
}
