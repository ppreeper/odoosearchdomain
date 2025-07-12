package odoosearchdomain

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var (
	ErrSyntax              = errors.New("invalid syntax")
	ErrInvalidComparator   = errors.New("invalid comparator")
	ErrInvalidTermValues   = errors.New("invalid term values")
	ErrNotEnoughAndOrTerms = errors.New("not enough AND/OR terms")
	ErrNotEnoughNotTerms   = errors.New("not enough NOT terms")
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
	// new checks
	emptySquareBrackets = regexp.MustCompile(`\[\s*\]`)
	termSplitter        = regexp.MustCompile(`\s*\(.+?\)\s*?|!|&|\|`)
	WithinSquareBracket = regexp.MustCompile(`\[\s*.?\s*\]|\[\s*.*\s*\]`)

	hasList   = regexp.MustCompile(`\(\s*.+\[.+?\]\s*\)`)
	tokenizer = regexp.MustCompile(`\(\s*(.+?)\s*\)|!|&|\|`)
)

func Parser(domain string) (filter []any, err error) {
	domain = strings.TrimSpace(domain)
	if domain == "" || emptySquareBrackets.MatchString(domain) {
		return []any{}, nil
	}
	if !WithinSquareBracket.MatchString(domain) {
		return []any{}, ErrSyntax
	}
	domain = strings.TrimSpace(strings.Trim(domain, "[]"))
	// fmt.Println("Parsing domain:", domain)

	if len(hasList.FindAllString(domain, -1)) == 0 && len(tokenizer.FindAllString(domain, -1)) == 0 {
		return []any{}, ErrSyntax
	}

	if hasList.MatchString(domain) {
		return []any{"has list"}, nil
	}

	if tokenizer.MatchString(domain) {
		tokens := tokenizer.FindAllString(domain, -1)
		pTokens(tokens)
		return parseTokens(domain)
	}

	// ts := tokenizer.FindAllString(domain, -1)
	// fmt.Printf("Tokens found: %v\n", ts)
	// if !tokenizer.MatchString(domain) {
	// 	return []any{}, ErrSyntax
	// }

	return []any{"not parsed"}, nil
}

func pTokens(tokens []string) {
}

func parseTokens(domain string) (filter []any, err error) {
	tokens := tokenizer.FindAllString(domain, -1)
	for i, token := range tokens {
		token = strings.TrimSpace(token)
		// filter = append(filter, token)
		fmt.Println("token:", token)
		if token == "!" {
			// naive check for NOT terms
			if i+1 > len(tokens)-1 {
				return []any{}, ErrNotEnoughNotTerms
			}
			fmt.Println("Found NOT operator", tokens[i+1:])
			_, err := checkNot(tokens[i+1:])
			if err != nil {
				return []any{}, err
			}
			continue
		}
		if token == "|" {
			// naive check for AND/OR terms
			if i+2 > len(tokens)-1 {
				return []any{}, ErrNotEnoughAndOrTerms
			}
			fmt.Println("Found OR operator", tokens[i+1:])
			continue
		}
		if token == "&" {
			// naive check for AND/OR terms
			if i+2 > len(tokens)-1 {
				return []any{}, ErrNotEnoughAndOrTerms
			}
			fmt.Println("Found AND operator", tokens[i+1:])
			continue
		}
		termArg := termArg.FindStringSubmatch(token)
		if len(termArg) > 0 {
			fmt.Println("Found term argument:", termArg, "len:", len(termArg))
			fterm, err := getTerm(termArg)
			if err != nil {
				return []any{}, err
			}
			filter = append(filter, fterm)
		}
	}
	return filter, nil
}

func checkNot(terms []string) ([]any, error) {
	for _, term := range terms {
		fmt.Println("Checking term:", term)
	}
	// if index+1 > len(terms)-1 {
	// 	return false, ErrNotEnoughNotTerms
	// }
	// if _, ok := terms[index+1].([]any); !ok {
	// 	return false, ErrNotEnoughNotTerms
	// }
	return []any{}, nil
}

func ParseDomain(domain string) (filter []any, err error) {
	if domain == "" {
		return []any{}, ErrSyntax
	}

	if !startEnd.MatchString(domain) {
		return []any{}, ErrSyntax
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
		return nil, ErrInvalidComparator
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
		return nil, ErrInvalidTermValues
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
				return []any{}, ErrNotEnoughAndOrTerms
			}
			if !checkIfTerm(terms, i, 2) {
				return []any{}, ErrNotEnoughAndOrTerms
			}
		case "!":
			if i+1 > len(terms)-1 {
				return []any{}, ErrNotEnoughNotTerms
			}
			if !checkIfTerm(terms, i, 1) {
				return []any{}, ErrNotEnoughNotTerms
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

func Parse(domain string) (filter []any, err error) {
	domain = strings.TrimSpace(domain)
	if domain == "" || emptySquareBrackets.MatchString(domain) {
		return []any{}, nil
	}
	if !WithinSquareBracket.MatchString(domain) {
		return []any{}, ErrSyntax
	}
	// domain = strings.TrimSpace(strings.Trim(domain, "[]"))
	// err = limitSplicer(domain)
	// if err != nil {
	// 	return []any{}, err
	// }
	tokenize(domain)

	return []any{"not parsed"}, nil
}

func tokenize(domain string) {
	SquareBrackets := [][]int{}
	SquareBracketsTemp := [][]int{}
	// SquareBracketsIndex := 0
	SquareBracketsCount := 0
	Brackets := [][]int{}
	BracketsTemp := [][]int{}
	// BracketsIndex := 0
	BracketsCount := 0
	Quotes := []int{}
	Commas := []int{}
	for i, s := range domain {
		switch s {
		case '\'':
			Quotes = append(Quotes, i)
		case ',':
			Commas = append(Commas, i)
		case '[':
			SquareBracketsTemp = append(SquareBracketsTemp, []int{i + 1, 0})
			// SquareBracketsIndex++
			SquareBracketsCount++
		case ']':
			if SquareBracketsCount == 0 {
				fmt.Println("Unmatched right bracket at index:", i)
				return
			}
			SquareBracketsTemp[len(SquareBracketsTemp)-1][1] = i
			SquareBrackets = append(SquareBrackets, SquareBracketsTemp[len(SquareBracketsTemp)-1])
			SquareBracketsTemp = SquareBracketsTemp[:len(SquareBracketsTemp)-1]
			// SquareBracketsTemp = append(SquareBracketsTemp[:SquareBrackets
			// SquareBracketsIndex--
			SquareBracketsCount--
		case '(':
			BracketsTemp = append(BracketsTemp, []int{i + 1, 0})
			// BracketsIndex++
			BracketsCount++
		case ')':
			if BracketsCount == 0 {
				fmt.Println("Unmatched right brace at index:", i)
				return
			}
			BracketsTemp[len(BracketsTemp)-1][1] = i
			Brackets = append(Brackets, BracketsTemp[len(BracketsTemp)-1])
			BracketsTemp = BracketsTemp[:len(BracketsTemp)-1]
			// BracketsIndex--
			BracketsCount--
		}
	}
	fmt.Println("----------")
	fmt.Println("Tokens with square brackets:", SquareBrackets)
	fmt.Println("square brackets count:", SquareBracketsCount)
	fmt.Println()
	for _, v := range SquareBrackets {
		fmt.Printf("Square brackets token:>%s<\n", domain[v[0]:v[1]])
	}
	fmt.Println()
	fmt.Println("----------")
	fmt.Println("Tokens with brackets:", Brackets)
	fmt.Println("brackets count:", BracketsCount)
	fmt.Println()
	for _, v := range Brackets {
		fmt.Printf("Brackets token:>%s<\n", domain[v[0]:v[1]])
	}
	fmt.Println()
	fmt.Println("----------")
	fmt.Println()
	fmt.Println("Single quotes:", Quotes)
	fmt.Println()
	fmt.Println("----------")
	fmt.Println()
	fmt.Println("Commas:", Commas)
	fmt.Println()
	fmt.Println("----------")
	fmt.Println()
}

func limitSplicer(domain string) error {
	// if err := limitCounter(domain); err != nil {
	// 	return err
	// }

	limits := Limits{}
	limits.limitLists(domain)

	if len(limits.Commas) < 2 {
		return fmt.Errorf("not enough commas in domain: %s", domain)
	}
	fmt.Println("Domain length:", len(domain))
	fmt.Println()

	fmt.Println("----------")
	fmt.Println("Left brackets:", limits.LeftBrackets)
	fmt.Println("Right brackets:", limits.RightBrackets)
	tS := append(limits.LeftBrackets, limits.RightBrackets...)
	sort.Ints(tS)
	fmt.Println("Sorted brackets:", tS)

	tokensS := [][]int{}
	for _, v := range limits.LeftBrackets {
		tokensS = append(tokensS, []int{v + 1, 0})
	}
	fmt.Println("Tokens with square brackets:", tokensS)

	for _, v := range limits.RightBrackets {
		for j, t := range tokensS {
			if v > t[0] && t[1] == 0 {
				tokensS[j][1] = v
				break
			}
		}
	}

	fmt.Println("Tokens with square brackets:", tokensS)
	fmt.Println()

	fmt.Println("----------")
	fmt.Println("Left braces:", limits.LeftBraces)
	fmt.Println("Right braces:", limits.RightBraces)
	// tB := append(limits.LeftBraces, limits.RightBraces...)
	// sort.Ints(tB)
	// fmt.Println("Sorted braces:", tB)

	// tokensB := [][]int{}
	// for _, v := range limits.LeftBraces {
	// 	tokensB = append(tokensB, []int{v + 1, 0})
	// }

	// for _, v := range limits.RightBraces {
	// 	for j, t := range tokensS {
	// 		if v > t[0] && t[1] == 0 {
	// 			tokensS[j][1] = v
	// 			break
	// 		}
	// 	}
	// }

	// fmt.Println("Tokens with braces:", tokensB)
	fmt.Println()

	fmt.Println("----------")
	fmt.Println("Single quotes:", limits.SingleQuotes)

	tokensQ := [][]int{}
	for i := 0; i < len(limits.SingleQuotes); i += 2 {
		tokensQ = append(tokensQ, []int{limits.SingleQuotes[i] + 1, limits.SingleQuotes[i+1]})
	}
	fmt.Println("Tokens with single quotes:", tokensQ)
	fmt.Println()

	fmt.Println("----------")
	fmt.Println("Commas:", limits.Commas)
	// tokensC := [][]int{}
	// for i, v := range limits.Commas {
	// 	if i == 0 {
	// 		tokensC = append(tokensC, []int{0, v})
	// 	} else {
	// 		tokensC = append(tokensC, []int{limits.Commas[i-1], v})
	// 	}
	// 	if i == len(limits.Commas)-1 {
	// 		tokensC = append(tokensC, []int{v, len(domain)})
	// 	}
	// }
	// fmt.Println("Tokens with commas:", tokensC)
	fmt.Println()

	// for _, v := range tokensS {
	// 	fmt.Printf("tokenS:>%s<\n", domain[v[0]:v[1]])
	// }
	// fmt.Println()
	// for _, v := range tokensB {
	// 	fmt.Printf("tokenB:>%s<\n", domain[v[0]:v[1]])
	// }
	// fmt.Println()
	// for _, v := range tokensQ {
	// 	fmt.Printf("tokenQ:>%s<\n", domain[v[0]:v[1]])
	// }
	// fmt.Println()
	// for _, v := range tokensC {
	// 	fmt.Printf("tokenC:>%s<\n", domain[v[0]:v[1]])
	// }
	// fmt.Println()

	fmt.Println()
	// tokens := []string{}
	// for i := 0; i < len(limits.SingleQuotes); i += 2 {
	// 	token := domain[limits.SingleQuotes[i]+1 : limits.SingleQuotes[i+1]]
	// 	tokens = append(tokens, token)
	// 	fmt.Printf("token:>%s<\n", token)
	// }

	// bracelist := [][]int{}
	// for i, l := range lBrace {
	// 	if i >= len(rBrace) {
	// 		// fmt.Printf("Left brace at %d has no matching right brace\n", l)
	// 		continue
	// 	}
	// 	for j, r := range rBrace {
	// 		if l < r {
	// 			// fmt.Printf("Left brace at %d has matching right brace at %d\n", l, r)
	// 			bracelist = append(bracelist, []int{l, r})
	// 			break
	// 		}
	// 		if j == len(rBrace)-1 {
	// 			// fmt.Printf("Left brace at %d has no matching right brace\n", l)
	// 		}
	// 	}
	// }
	// fmt.Println("Bracelist:", bracelist)
	// for _, braces := range bracelist {
	// 	// check for short slices
	// 	if braces[1]-braces[0] < 2 {
	// 		return fmt.Errorf("braces at %d and %d are too close together", braces[0], braces[1])
	// 	}
	// }

	return nil
}

type Limits struct {
	LeftBraces    []int
	RightBraces   []int
	LeftBrackets  []int
	RightBrackets []int
	SingleQuotes  []int
	Commas        []int
}

// func (l *Limits) limitCounter(domain string) error {
// 	limitCounts := []int{0, 0, 0, 0, 0} // (,),[,],'
// 	for _, r := range domain {
// 		switch r {
// 		case '(':
// 			limitCounts[0]++
// 		case ')':
// 			limitCounts[1]++
// 		case '[':
// 			limitCounts[2]++
// 		case ']':
// 			limitCounts[3]++
// 		case '\'':
// 			limitCounts[4]++
// 		default:
// 			continue
// 		}
// 	}
// 	if limitCounts[0] != limitCounts[1] || limitCounts[2] != limitCounts[3] || limitCounts[4]%2 != 0 {
// 		return fmt.Errorf("mismatched brackets or parentheses in domain: %s", domain)
// 	}
// 	return nil
// }

func (l *Limits) limitLists(domain string) {
	for i, r := range domain {
		switch r {
		case '(':
			l.LeftBraces = append(l.LeftBraces, i)
		case ')':
			l.RightBraces = append(l.RightBraces, i)
		case '[':
			l.LeftBrackets = append(l.LeftBrackets, i)
		case ']':
			l.RightBrackets = append(l.RightBrackets, i)
		case '\'':
			l.SingleQuotes = append(l.SingleQuotes, i)
		case ',':
			l.Commas = append(l.Commas, i)
		default:
			continue
		}
	}
}
