package odoosearchdomain

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
)

var (
	errSyntax              = errors.New("invalid syntax")
	errInvalidComparator   = errors.New("invalid comparator")
	errInvalidTermValues   = errors.New("invalid term values")
	errNotEnoughAndOrTerms = errors.New("not enough AND/OR terms")
	errNotEnoughNotTerms   = errors.New("not enough NOT terms")
)

var (
	startEnd       = regexp.MustCompile(`\[\s*\(.+\)\s*\]`)
	termArg        = regexp.MustCompile(`\(\s*['"](.+?)['"]\s*,\s*['"](.+?)['"]\s*,\s*(.+)\s*\)`)
	comparatorTerm = regexp.MustCompile(`(?:=|!=|>|>=|<|<=|=\?|like|not like|ilike|not ilike|=ilike|in|not in|child_of|parent_of|any|not any)`)
	splitTerms     = regexp.MustCompile(`\([^()]*?(?:\([^()]*?\)[^()]*?)*?\)|!|&|\|`)
	intTerm        = regexp.MustCompile(`^\d+?$`)
	floatTerm      = regexp.MustCompile(`^\d+\.\d+$`)
	listTerm       = regexp.MustCompile(`\[.+?\]`)
	termSplits     = regexp.MustCompile(`\s*\(.+?\)\s*?`)
)

func ParseDomain(domain string) (filter []any, err error) {
	if domain == "" {
		return []any{}, errSyntax
	}

	if !startEnd.MatchString(domain) {
		return []any{}, errSyntax
	}

	results := splitTerms.FindAllString(domain, -1)
	for _, result := range results {
		switch result {
		case "!":
			filter = append(filter, "!")
		case "|":
			filter = append(filter, "|")
		case "&":
			filter = append(filter, "&")
		default:
			if termArg.MatchString(result) {
				resreg := termArg.FindAllStringSubmatch(result, -1)

				if len(resreg[0]) > 0 {
					fterm, err := getTerm(resreg[0])
					if err != nil {
						return []any{}, err
					}
					filter = append(filter, fterm)
				}
			}
		}
	}
	return checkAndOrTerms(filter)
}

func getTerm(terms []string) ([]any, error) {
	field := strings.TrimSpace(terms[1])
	comparator := strings.TrimSpace(terms[2])
	if !comparatorTerm.MatchString(comparator) {
		return nil, errInvalidComparator
	}
	var value any
	switch strings.TrimSpace(terms[3]) {
	case "True":
		value = true
	case "False":
		value = false
	case "None":
		value = nil
	default:
		if strings.HasPrefix(terms[3], "'") && strings.HasSuffix(terms[3], "'") {
			value = strings.TrimSpace(strings.Trim(terms[3], "'"))
			break
		} else if intTerm.MatchString(strings.TrimSpace(terms[3])) {
			if num, err := strconv.Atoi(strings.TrimSpace(terms[3])); err == nil {
				value = num
				break
			}
		} else if floatTerm.MatchString(strings.TrimSpace(terms[3])) {
			if num, err := strconv.ParseFloat(strings.TrimSpace(terms[3]), 64); err == nil {
				value = num
				break
			}
		} else if listTerm.MatchString(strings.TrimSpace(terms[3])) {
			terms[3] = strings.TrimSpace(strings.Trim(terms[3], "[]"))
			value = getList(terms[3])
		} else {
			value = terms[3]
		}
	}

	if field == "" || comparator == "" || value == "" {
		return nil, errInvalidTermValues
	}

	return []any{field, comparator, value}, nil
}

func getList(content string) []any {
	var results []any
	if strings.Contains(content, "(") {
		termSplits := termSplits.FindAllStringSubmatch(content, -1)
		for _, part := range termSplits {
			if termArg.MatchString(part[0]) {
				resreg := termArg.FindAllStringSubmatch(part[0], -1)
				if len(resreg[0]) > 0 {
					fterm, err := getTerm(resreg[0])
					if err != nil {
						return []any{}
					}
					results = append(results, fterm)
				}
			}
		}
	} else {
		for part := range strings.SplitSeq(content, ",") {
			value := strings.TrimSpace(strings.Trim(part, "'"))
			if intTerm.MatchString(value) {
				if num, err := strconv.Atoi(value); err == nil {
					results = append(results, num)
					continue
				}
			} else if floatTerm.MatchString(value) {
				if num, err := strconv.ParseFloat(value, 64); err == nil {
					results = append(results, num)
					continue
				}
			} else {
				results = append(results, strings.TrimSpace(strings.Trim(part, "'")))
			}
		}
	}
	return results
}

func checkAndOrTerms(terms []any) (results []any, err error) {
	for i, term := range terms {
		switch term {
		case "&", "|":
			if i+2 > len(terms)-1 {
				return []any{}, errNotEnoughAndOrTerms
			}
			if !checkIfTerm(terms, i, 2) {
				return []any{}, errNotEnoughAndOrTerms
			}
		case "!":
			if i+1 > len(terms)-1 {
				return []any{}, errNotEnoughNotTerms
			}
			if !checkIfTerm(terms, i, 1) {
				return []any{}, errNotEnoughNotTerms
			}
		}
	}
	return terms, nil
}

func checkIfTerm(terms []any, index int, arrity int) bool {
	allValid := true
	for i := index + 1; i <= index+arrity; i++ {
		switch v := terms[i].(type) {
		case []any:
			if len(v) != 3 {
				allValid = false
			}
		default:
			allValid = false
		}
	}
	return allValid
}

func Fields(field string) []string {
	if field != "" {
		return strings.Split(field, ",")
	} else {
		return []string{}
	}
}
