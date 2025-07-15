package odoosearchdomain

import (
	"errors"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

var ErrSyntax = errors.New("invalid syntax")

// ErrInvalidComparator   = errors.New("invalid comparator")
// ErrInvalidTermValues   = errors.New("invalid term values")
// ErrNotEnoughAndOrTerms = errors.New("not enough AND/OR terms")
// ErrNotEnoughNotTerms   = errors.New("not enough NOT terms")

var (
	withinSquareBracket = regexp.MustCompile(`\[\s*.?\s*\]|\[\s*.*\s*\]`)
	comparatorTerm      = regexp.MustCompile(`(?:=|!=|>|>=|<|<=|=\?|like|not like|ilike|not ilike|=ilike|in|not in|child_of|parent_of|any|not any)`)
	emptySquareBrackets = regexp.MustCompile(`\[\s*\]`)
	notNode             = regexp.MustCompile(`'!'`)
	orNode              = regexp.MustCompile(`'\|'`)
	andNode             = regexp.MustCompile(`'&'`)
	termOneNode         = regexp.MustCompile(`\((\s*'.+?'\s*,\s*'.+?'\s*,\s*.+?\s*)\)$`)
	oneQuote            = regexp.MustCompile(`'(.+?)'`)
	trueFalseNone       = regexp.MustCompile(`(True|False|None|true|false|none)`)
	nonQuote            = regexp.MustCompile(`,\s*(.+)?`)
	listItems           = regexp.MustCompile(`\[(.+?)\]$`)
	floatTerm           = regexp.MustCompile(`(\d+\.\d+)`)
	intTerm             = regexp.MustCompile(`(\d+)`)
)

func ParseDomain(domain string) (filter []any, err error) {
	domain = strings.TrimSpace(domain)
	if domain == "" || emptySquareBrackets.MatchString(domain) {
		return []any{}, nil
	}
	// if !withinSquareBracket.MatchString(domain) {
	// 	return []any{}, ErrSyntax
	// }

	filter, err = tokenIter(domain)
	if err != nil {
		return []any{}, err
	}

	return filter, nil
}

func tokenIter(domain string) ([]any, error) {
	newdomain := strings.Trim(domain, "[]")
	dom := []any{}
	bracketcount := 0
	sub := ""
	for _, s := range newdomain {
		sub += string(s)
		switch s {
		case '(':
			bracketcount++
		case ')':
			bracketcount--
		}
		if notNode.MatchString(sub) {
			dom = append(dom, "!")
			sub = ""
		}
		if orNode.MatchString(sub) {
			dom = append(dom, "|")
			sub = ""
		}
		if andNode.MatchString(sub) {
			dom = append(dom, "&")
			sub = ""
		}
		if termOneNode.MatchString(sub) && bracketcount == 0 {
			term := termOneNode.FindStringSubmatch(sub)
			result, err := termIter(term[1])
			if err != nil {
				return []any{}, ErrSyntax
			}
			dom = append(dom, result)
			sub = ""
		}
	}
	return dom, nil
}

func termIter(domain string) (terms []any, err error) {
	parts := oneQuote.FindAllStringSubmatch(domain, -1)
	indexes := oneQuote.FindAllStringIndex(domain, -1)

	element := parts[0][1]
	comparator := parts[1][1]
	if element == "" || comparator == "" {
		return []any{}, ErrSyntax
	}
	if !comparatorTerm.MatchString(comparator) {
		return []any{}, ErrSyntax
	}
	terms = append(terms, element, comparator)

	valueDomain := domain[indexes[1][1]:]

	if listItems.MatchString(valueDomain) {
		values := listItems.FindStringSubmatch(valueDomain)[1]
		v, e := tokenIter(valueDomain)
		if e != nil {
			return []any{}, e
		}
		if len(v) != 0 {
			terms = append(terms, v)
			return terms, nil
		} else if floatTerm.MatchString(values) {
			floatValues := floatTerm.FindAllStringSubmatch(values, -1)
			vals := []any{}
			for _, v := range floatValues {
				if num, err := strconv.ParseFloat(v[1], 64); err == nil {
					vals = append(vals, num)
				}
			}
			terms = append(terms, vals)
			return terms, nil
		} else if intTerm.MatchString(values) {
			intValues := intTerm.FindAllStringSubmatch(values, -1)
			vals := []any{}
			for _, v := range intValues {
				if num, err := strconv.Atoi(v[1]); err == nil {
					vals = append(vals, num)
				}
			}
			terms = append(terms, vals)
			return terms, nil
		} else if oneQuote.MatchString(values) {
			stringValues := oneQuote.FindAllStringSubmatch(values, -1)
			vals := []any{}
			for _, v := range stringValues {
				vals = append(vals, v[1])
			}
			terms = append(terms, vals)
			return terms, nil
		}
		return
	} else if oneQuote.MatchString(valueDomain) {
		value := oneQuote.FindStringSubmatch(valueDomain)[1]
		terms = append(terms, value)
	} else if floatTerm.MatchString(valueDomain) {
		value := floatTerm.FindStringSubmatch(valueDomain)[1]
		if num, err := strconv.ParseFloat(value, 64); err == nil {
			terms = append(terms, num)
		}
	} else if intTerm.MatchString(valueDomain) {
		value := intTerm.FindStringSubmatch(valueDomain)[1]
		if num, err := strconv.Atoi(value); err == nil {
			terms = append(terms, num)
		}
	} else if trueFalseNone.MatchString(valueDomain) {
		value := trueFalseNone.FindStringSubmatch(valueDomain)[1]
		switch value {
		case "True", "true":
			terms = append(terms, true)
		case "False", "false":
			terms = append(terms, false)
		case "None", "none":
			terms = append(terms, nil)
		}
	} else if nonQuote.MatchString(valueDomain) {
		value := nonQuote.FindStringSubmatch(valueDomain)[1]
		terms = append(terms, value)
	}
	return terms, nil
}

func validateDomain(domain []any) error {
	if len(domain) == 0 {
		return ErrSyntax
	}
	if len(domain) == 1 && reflect.TypeOf(domain[0]).Kind() != reflect.Slice {
		return ErrSyntax
	}
	if len(domain) > 1 && reflect.TypeOf(domain[0]).Kind() != reflect.Slice {
		return ErrSyntax
	}
	for _, term := range domain {
		if reflect.TypeOf(term).Kind() != reflect.Slice && term != "!" && term != "|" && term != "&" {
			return ErrSyntax
		}
	}
	return nil
}
