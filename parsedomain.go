package odoosearchdomain

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// Sentinel errors for domain parsing and validation.
var (
	ErrSyntax              = errors.New("invalid syntax")
	ErrNotEnoughAndOrTerms = errors.New("not enough AND/OR terms")
	ErrNotEnoughNotTerms   = errors.New("not enough NOT terms")
)

// ============================================================
// Phase 1: Lexer — single-pass tokenizer, O(n)
// ============================================================

type tokenType int

const (
	tokEOF      tokenType = iota
	tokLBracket           // [
	tokRBracket           // ]
	tokLParen             // (
	tokRParen             // )
	tokComma              // ,
	tokString             // 'abc' or "abc"
	tokInt                // 123, -5
	tokFloat              // 1.23, -0.5, 1e+21
	tokTrue               // True, true
	tokFalse              // False, false
	tokNone               // None, none
)

type token struct {
	typ tokenType
	str string // for strings: unquoted content; for numbers: raw digits
	pos int    // byte position in input (for error messages)
}

type lexer struct {
	input []rune
	pos   int
}

func newLexer(input string) *lexer {
	return &lexer{input: []rune(input)}
}

func (l *lexer) peek() rune {
	if l.pos >= len(l.input) {
		return 0
	}
	return l.input[l.pos]
}

func (l *lexer) advance() rune {
	r := l.input[l.pos]
	l.pos++
	return r
}

func (l *lexer) skipWhitespace() {
	for l.pos < len(l.input) && unicode.IsSpace(l.input[l.pos]) {
		l.pos++
	}
}

func (l *lexer) tokenize() ([]token, error) {
	var tokens []token
	for {
		l.skipWhitespace()
		if l.pos >= len(l.input) {
			tokens = append(tokens, token{typ: tokEOF, pos: l.pos})
			return tokens, nil
		}

		r := l.peek()
		startPos := l.pos

		switch {
		case r == '[':
			l.advance()
			tokens = append(tokens, token{typ: tokLBracket, pos: startPos})
		case r == ']':
			l.advance()
			tokens = append(tokens, token{typ: tokRBracket, pos: startPos})
		case r == '(':
			l.advance()
			tokens = append(tokens, token{typ: tokLParen, pos: startPos})
		case r == ')':
			l.advance()
			tokens = append(tokens, token{typ: tokRParen, pos: startPos})
		case r == ',':
			l.advance()
			tokens = append(tokens, token{typ: tokComma, pos: startPos})
		case r == '\'' || r == '"':
			tok, err := l.lexString(r)
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, tok)
		case r == '-' || unicode.IsDigit(r):
			tok, err := l.lexNumber()
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, tok)
		case unicode.IsLetter(r):
			tok, err := l.lexKeyword()
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, tok)
		default:
			return nil, fmt.Errorf("%w: unexpected character %q at position %d", ErrSyntax, string(r), l.pos)
		}
	}
}

// lexString reads a single-quoted or double-quoted string, decoding backslash
// escapes the way Python's repr() encodes them (\n, \t, \xNN, \uNNNN, ...).
func (l *lexer) lexString(quote rune) (token, error) {
	startPos := l.pos
	l.advance() // skip opening quote
	var sb strings.Builder
	for l.pos < len(l.input) {
		r := l.advance()
		if r == '\\' && l.pos < len(l.input) {
			s, err := l.unescape(l.advance())
			if err != nil {
				return token{}, err
			}
			sb.WriteString(s)
			continue
		}
		if r == quote {
			return token{typ: tokString, str: sb.String(), pos: startPos}, nil
		}
		sb.WriteRune(r)
	}
	return token{}, fmt.Errorf("%w: unterminated string starting at position %d", ErrSyntax, startPos)
}

// unescape decodes the rune following a backslash, consuming additional input
// for multi-character escapes (\xNN, \uNNNN, \UNNNNNNNN). Unknown escapes are
// passed through as a literal backslash plus the rune. Out-of-range and
// surrogate code points are rejected rather than silently replaced.
func (l *lexer) unescape(r rune) (string, error) {
	switch r {
	case '\'', '"', '\\':
		return string(r), nil
	case 'n':
		return "\n", nil
	case 't':
		return "\t", nil
	case 'r':
		return "\r", nil
	case 'b':
		return "\b", nil
	case 'f':
		return "\f", nil
	case 'v':
		return "\v", nil
	case 'a':
		return "\a", nil
	case 'x':
		if v, ok := l.readHex(2); ok {
			return string(rune(v)), nil
		}
	case 'u':
		if v, ok := l.readHex(4); ok {
			return encodedRune(v, l.pos)
		}
	case 'U':
		if v, ok := l.readHex(8); ok {
			return encodedRune(v, l.pos)
		}
	}
	return "\\" + string(r), nil
}

// encodedRune converts a decoded escape code point into a string, rejecting
// values that are not Unicode scalar values (out of range or surrogates).
func encodedRune(v int64, pos int) (string, error) {
	if v > unicode.MaxRune || (v >= 0xD800 && v <= 0xDFFF) {
		return "", fmt.Errorf("%w: invalid Unicode code point U+%04X at position %d", ErrSyntax, v, pos)
	}
	return string(rune(v)), nil
}

// readHex reads n hex digits and returns their value, or false if fewer than
// n hex digits remain. On failure the input position is left unchanged.
func (l *lexer) readHex(n int) (int64, bool) {
	startPos := l.pos
	var v int64
	for i := 0; i < n; i++ {
		if l.pos >= len(l.input) {
			l.pos = startPos
			return 0, false
		}
		d := hexDigit(l.input[l.pos])
		if d < 0 {
			l.pos = startPos
			return 0, false
		}
		v = v*16 + int64(d)
		l.advance()
	}
	return v, true
}

// hexDigit returns the value of a hexadecimal digit rune, or -1 if not a digit.
func hexDigit(r rune) int {
	switch {
	case r >= '0' && r <= '9':
		return int(r - '0')
	case r >= 'a' && r <= 'f':
		return int(r-'a') + 10
	case r >= 'A' && r <= 'F':
		return int(r-'A') + 10
	default:
		return -1
	}
}

// lexNumber reads an integer or float: an optional minus sign, digits, an
// optional decimal fraction, and an optional base-10 exponent (1e+21, 1.5E-3).
// Python's repr() uses scientific notation for large and small floats.
func (l *lexer) lexNumber() (token, error) {
	startPos := l.pos
	var sb strings.Builder

	if l.peek() == '-' {
		sb.WriteRune(l.advance())
	}

	if l.pos >= len(l.input) || !unicode.IsDigit(l.peek()) {
		return token{}, fmt.Errorf("%w: expected digit after '-' at position %d", ErrSyntax, startPos)
	}

	for l.pos < len(l.input) && unicode.IsDigit(l.peek()) {
		sb.WriteRune(l.advance())
	}

	isFloat := false

	// Decimal point → float
	if l.pos < len(l.input) && l.peek() == '.' {
		sb.WriteRune(l.advance())
		if l.pos >= len(l.input) || !unicode.IsDigit(l.peek()) {
			return token{}, fmt.Errorf("%w: expected digit after '.' at position %d", ErrSyntax, startPos)
		}
		for l.pos < len(l.input) && unicode.IsDigit(l.peek()) {
			sb.WriteRune(l.advance())
		}
		isFloat = true
	}

	// Exponent → float (e.g. 1e+21, 1.5e-3)
	if l.pos < len(l.input) && (l.peek() == 'e' || l.peek() == 'E') {
		savePos := l.pos
		var exp strings.Builder
		exp.WriteRune(l.advance())
		if l.pos < len(l.input) && (l.peek() == '+' || l.peek() == '-') {
			exp.WriteRune(l.advance())
		}
		if l.pos >= len(l.input) || !unicode.IsDigit(l.peek()) {
			// Not an exponent after all (e.g. "1edge" is 1 + keyword "edge").
			l.pos = savePos
		} else {
			for l.pos < len(l.input) && unicode.IsDigit(l.peek()) {
				exp.WriteRune(l.advance())
			}
			sb.WriteString(exp.String())
			isFloat = true
		}
	}

	if isFloat {
		return token{typ: tokFloat, str: sb.String(), pos: startPos}, nil
	}
	return token{typ: tokInt, str: sb.String(), pos: startPos}, nil
}

// lexKeyword reads an alphabetic identifier and maps it to True/False/None.
func (l *lexer) lexKeyword() (token, error) {
	startPos := l.pos
	var sb strings.Builder
	for l.pos < len(l.input) && (unicode.IsLetter(l.peek()) || l.peek() == '_') {
		sb.WriteRune(l.advance())
	}
	word := sb.String()
	switch word {
	case "True", "true":
		return token{typ: tokTrue, str: word, pos: startPos}, nil
	case "False", "false":
		return token{typ: tokFalse, str: word, pos: startPos}, nil
	case "None", "none":
		return token{typ: tokNone, str: word, pos: startPos}, nil
	default:
		return token{}, fmt.Errorf("%w: unexpected keyword %q at position %d", ErrSyntax, word, startPos)
	}
}

// ============================================================
// Phase 2: Recursive Descent Parser
// ============================================================

// maxNestingDepth bounds recursion during parsing and validation. Domains
// that nest deeper (lists within lists, or chains of prefix operators) are
// rejected with ErrSyntax instead of exhausting the stack.
const maxNestingDepth = 1000

// validComparators is the complete set of Odoo domain operators.
var validComparators = map[string]bool{
	"=": true, "!=": true, ">": true, ">=": true, "<": true, "<=": true,
	"=?": true, "like": true, "not like": true, "ilike": true, "not ilike": true,
	"=like": true, "=ilike": true, "in": true, "not in": true,
	"child_of": true, "parent_of": true, "any": true, "not any": true,
}

type parser struct {
	tokens []token
	pos    int
	depth  int // current bracket-nesting depth (guarded by maxNestingDepth)
}

func newParser(tokens []token) *parser {
	return &parser{tokens: tokens}
}

func (p *parser) peek() token {
	if p.pos >= len(p.tokens) {
		return token{typ: tokEOF}
	}
	return p.tokens[p.pos]
}

func (p *parser) peekAt(offset int) token {
	idx := p.pos + offset
	if idx >= len(p.tokens) {
		return token{typ: tokEOF}
	}
	return p.tokens[idx]
}

func (p *parser) advance() token {
	t := p.tokens[p.pos]
	p.pos++
	return t
}

func (p *parser) expect(typ tokenType) (token, error) {
	t := p.peek()
	if t.typ != typ {
		return t, fmt.Errorf("%w: unexpected token at position %d", ErrSyntax, t.pos)
	}
	return p.advance(), nil
}

// parseDomain parses: '[' items ']' | '[' ']' | '[' '(' ')' ']'
func (p *parser) parseDomain() ([]any, error) {
	if _, err := p.expect(tokLBracket); err != nil {
		return nil, err
	}

	// Empty: []
	if p.peek().typ == tokRBracket {
		p.advance()
		return []any{}, nil
	}

	// Empty with parens: [()]
	if p.peek().typ == tokLParen && p.peekAt(1).typ == tokRParen && p.peekAt(2).typ == tokRBracket {
		p.advance() // (
		p.advance() // )
		p.advance() // ]
		return []any{}, nil
	}

	items, err := p.parseItems()
	if err != nil {
		return nil, err
	}

	if _, err := p.expect(tokRBracket); err != nil {
		return nil, err
	}

	return items, nil
}

// parseItems parses: item (',' item)*
func (p *parser) parseItems() ([]any, error) {
	var items []any

	item, err := p.parseItem()
	if err != nil {
		return nil, err
	}
	items = append(items, item...)

	for p.peek().typ == tokComma {
		p.advance() // consume comma
		// Allow trailing comma before ]
		if p.peek().typ == tokRBracket {
			break
		}
		item, err := p.parseItem()
		if err != nil {
			return nil, err
		}
		items = append(items, item...)
	}

	return items, nil
}

// parseItem parses a single domain element: either a connector ('&','|','!') or a term tuple.
// Returns a slice so connectors (1 element) and terms (1 element) can be appended uniformly.
func (p *parser) parseItem() ([]any, error) {
	t := p.peek()

	// Connector: a quoted string whose content is &, |, or !
	if t.typ == tokString && (t.str == "&" || t.str == "|" || t.str == "!") {
		p.advance()
		return []any{t.str}, nil
	}

	// Term tuple: ('field', 'operator', value)
	if t.typ == tokLParen {
		term, err := p.parseTerm()
		if err != nil {
			return nil, err
		}
		return []any{term}, nil
	}

	return nil, fmt.Errorf("%w: expected '(' or connector at position %d", ErrSyntax, t.pos)
}

// parseTerm parses: '(' STRING ',' STRING ',' value ')'
func (p *parser) parseTerm() ([]any, error) {
	if _, err := p.expect(tokLParen); err != nil {
		return nil, err
	}

	// Field name
	fieldTok, err := p.expect(tokString)
	if err != nil {
		return nil, fmt.Errorf("%w: expected field name string", ErrSyntax)
	}

	if _, err := p.expect(tokComma); err != nil {
		return nil, err
	}

	// Operator
	opTok, err := p.expect(tokString)
	if err != nil {
		return nil, fmt.Errorf("%w: expected operator string", ErrSyntax)
	}
	if !validComparators[opTok.str] {
		return nil, fmt.Errorf("%w: unknown operator %q", ErrSyntax, opTok.str)
	}

	if _, err := p.expect(tokComma); err != nil {
		return nil, err
	}

	// Value
	value, err := p.parseValue()
	if err != nil {
		return nil, err
	}

	if _, err := p.expect(tokRParen); err != nil {
		return nil, err
	}

	return []any{fieldTok.str, opTok.str, value}, nil
}

// parseValue parses: STRING | INT | FLOAT | TRUE | FALSE | NONE | list-or-domain
func (p *parser) parseValue() (any, error) {
	if p.depth >= maxNestingDepth {
		return nil, fmt.Errorf("%w: nesting exceeds %d levels", ErrSyntax, maxNestingDepth)
	}

	t := p.peek()

	switch t.typ {
	case tokString:
		p.advance()
		return t.str, nil
	case tokInt:
		p.advance()
		n, err := strconv.Atoi(t.str)
		if err == nil {
			return n, nil
		}
		// Python integers are unbounded; on int overflow fall back to float64.
		if errors.Is(err, strconv.ErrRange) {
			if f, ferr := strconv.ParseFloat(t.str, 64); ferr == nil {
				return f, nil
			}
		}
		return nil, fmt.Errorf("%w: invalid integer %q", ErrSyntax, t.str)
	case tokFloat:
		p.advance()
		f, err := strconv.ParseFloat(t.str, 64)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid float %q", ErrSyntax, t.str)
		}
		return f, nil
	case tokTrue:
		p.advance()
		return true, nil
	case tokFalse:
		p.advance()
		return false, nil
	case tokNone:
		p.advance()
		return nil, nil
	case tokLBracket:
		p.depth++
		v, err := p.parseListOrDomain()
		p.depth--
		if err != nil {
			return nil, err
		}
		return v, nil
	default:
		return nil, fmt.Errorf("%w: unexpected token in value position at %d", ErrSyntax, t.pos)
	}
}

// parseListOrDomain disambiguates between a nested domain [(...), ...] and a plain list [1,2,3].
// If the first element after [ is a tuple or a connector string, it tries domain
// parsing first. Nested domains are validated here, since only the parser knows
// which interpretation was chosen.
func (p *parser) parseListOrDomain() (any, error) {
	next := p.peekAt(1) // token after '['

	// A plain value list can never start with '(' — a term tuple is the only
	// reading, so propagate domain errors instead of masking them.
	if next.typ == tokLParen {
		domain, err := p.parseDomain()
		if err != nil {
			return nil, err
		}
		if err := validateTerms(domain, p.depth); err != nil {
			return nil, err
		}
		return domain, nil
	}

	// If the first element looks like a domain connector, try domain parsing.
	if next.typ == tokString && (next.str == "&" || next.str == "|" || next.str == "!") {
		savePos := p.pos
		domain, err := p.parseDomain()
		if err == nil {
			if verr := validateTerms(domain, p.depth); verr != nil {
				return nil, verr
			}
			return domain, nil
		}
		p.pos = savePos // backtrack: it was a plain list of strings
	}

	// Otherwise parse as a plain value list.
	return p.parseList()
}

// parseList parses: '[' value (',' value)* ']' | '[' ']'
func (p *parser) parseList() ([]any, error) {
	if _, err := p.expect(tokLBracket); err != nil {
		return nil, err
	}

	// Empty list
	if p.peek().typ == tokRBracket {
		p.advance()
		return []any{}, nil
	}

	var items []any
	val, err := p.parseValue()
	if err != nil {
		return nil, err
	}
	items = append(items, val)

	for p.peek().typ == tokComma {
		p.advance() // consume comma
		if p.peek().typ == tokRBracket {
			break // trailing comma
		}
		val, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		items = append(items, val)
	}

	if _, err := p.expect(tokRBracket); err != nil {
		return nil, err
	}

	return items, nil
}

// ============================================================
// Validation — prefix-notation arity checking
// ============================================================

// validateDomain walks the flat token list and checks that every & and | has
// 2 operands, and every ! has 1 operand, using proper recursive consumption.
func validateDomain(terms []any) ([]any, error) {
	if err := validateTerms(terms, 0); err != nil {
		return []any{}, err
	}
	return terms, nil
}

// validateTerms checks a complete term sequence at the given nesting depth.
// Nested domains held in term values are validated recursively.
func validateTerms(terms []any, depth int) error {
	pos := 0
	for pos < len(terms) {
		count, err := validateAt(terms, pos, depth)
		if err != nil {
			return err
		}
		pos += count
	}
	return nil
}

// looksLikeDomain reports whether v starts like a prefix-notation domain: a
// term tuple, or a connector followed by at least one term tuple. Requiring a
// tuple keeps plain string lists such as ['&', 'a'] out.
func looksLikeDomain(v []any) bool {
	if len(v) == 0 {
		return false
	}
	switch first := v[0].(type) {
	case []any:
		return true
	case string:
		if first != "&" && first != "|" && first != "!" {
			return false
		}
		for _, elem := range v[1:] {
			if _, ok := elem.([]any); ok {
				return true
			}
		}
	}
	return false
}

// validateAt validates the expression rooted at terms[pos] and returns
// the number of tokens consumed by that expression.
func validateAt(terms []any, pos, depth int) (int, error) {
	if depth >= maxNestingDepth {
		return 0, fmt.Errorf("%w: nesting exceeds %d levels", ErrSyntax, maxNestingDepth)
	}
	if pos >= len(terms) {
		return 0, fmt.Errorf("%w: unexpected end of domain", ErrSyntax)
	}

	switch terms[pos] {
	case "&", "|":
		// Binary operator: consumes 1 (itself) + operand1 + operand2
		if pos+1 >= len(terms) {
			return 0, ErrNotEnoughAndOrTerms
		}
		count1, err := validateAt(terms, pos+1, depth+1)
		if err != nil {
			return 0, err
		}
		if pos+1+count1 >= len(terms) {
			return 0, ErrNotEnoughAndOrTerms
		}
		count2, err := validateAt(terms, pos+1+count1, depth+1)
		if err != nil {
			return 0, err
		}
		return 1 + count1 + count2, nil

	case "!":
		// Unary operator: consumes 1 (itself) + operand
		if pos+1 >= len(terms) {
			return 0, ErrNotEnoughNotTerms
		}
		count, err := validateAt(terms, pos+1, depth+1)
		if err != nil {
			return 0, err
		}
		return 1 + count, nil

	default:
		// Leaf term (a []any tuple) — consumes 1. Validate any nested domain
		// held in the term's value.
		if term, ok := terms[pos].([]any); ok && len(term) == 3 {
			if nested, ok := term[2].([]any); ok && looksLikeDomain(nested) {
				if err := validateTerms(nested, depth+1); err != nil {
					return 0, err
				}
			}
		}
		return 1, nil
	}
}

// ============================================================
// Public API
// ============================================================

// ParseDomain parses an Odoo search domain string into a slice of terms and connectors.
//
// The domain string should be formatted according to the Odoo search domain syntax:
//
//	[('field','operator',value), '|', ('field2','op2',value2), ...]
//
// Input that is empty or lacks outer brackets is not treated as a domain and
// yields an empty slice with no error. Any input that begins with '[' must be
// a well-formed domain end to end: lexer, parser, and arity errors are all
// reported (ErrSyntax, ErrNotEnoughAndOrTerms, ErrNotEnoughNotTerms) rather
// than silently producing an empty result.
func ParseDomain(domain string) (filter []any, err error) {
	domain = strings.TrimSpace(domain)
	if domain == "" {
		return []any{}, nil
	}

	// Tokenize
	tokens, lexErr := newLexer(domain).tokenize()
	if lexErr != nil {
		// Input without brackets is not a domain — not an error.
		if !strings.HasPrefix(domain, "[") {
			return []any{}, nil
		}
		return []any{}, lexErr
	}

	// Domain must start with [
	if tokens[0].typ != tokLBracket {
		return []any{}, nil
	}

	// Parse
	p := newParser(tokens)
	result, parseErr := p.parseDomain()
	if parseErr != nil {
		return []any{}, parseErr
	}

	// Must have consumed all tokens (except trailing EOF)
	if p.peek().typ != tokEOF {
		return []any{}, fmt.Errorf("%w: unexpected trailing characters at position %d", ErrSyntax, p.peek().pos)
	}

	if len(result) == 0 {
		return []any{}, nil
	}

	// Validate prefix-notation structure
	return validateDomain(result)
}

// ValidateDomain checks parsed domain terms for correct AND/OR/NOT arity.
// Exported for backward compatibility.
func ValidateDomain(terms []any) (results []any, err error) {
	return validateDomain(terms)
}

// Fields splits a comma-separated field list into individual field names,
// trimming surrounding whitespace. Empty entries are dropped.
func Fields(field string) []string {
	if field == "" {
		return []string{}
	}
	parts := strings.Split(field, ",")
	fields := make([]string, 0, len(parts))
	for _, part := range parts {
		if name := strings.TrimSpace(part); name != "" {
			fields = append(fields, name)
		}
	}
	return fields
}
