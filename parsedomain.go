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
	tokFloat              // 1.23, -0.5
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

// lexString reads a single-quoted or double-quoted string with escape support.
func (l *lexer) lexString(quote rune) (token, error) {
	startPos := l.pos
	l.advance() // skip opening quote
	var sb strings.Builder
	for l.pos < len(l.input) {
		r := l.advance()
		if r == '\\' && l.pos < len(l.input) {
			next := l.advance()
			switch next {
			case '\'', '"', '\\':
				sb.WriteRune(next)
			default:
				sb.WriteRune('\\')
				sb.WriteRune(next)
			}
			continue
		}
		if r == quote {
			return token{typ: tokString, str: sb.String(), pos: startPos}, nil
		}
		sb.WriteRune(r)
	}
	return token{}, fmt.Errorf("%w: unterminated string starting at position %d", ErrSyntax, startPos)
}

// lexNumber reads an integer or float, optionally preceded by a minus sign.
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

	// Check for decimal point → float
	if l.pos < len(l.input) && l.peek() == '.' {
		sb.WriteRune(l.advance())
		if l.pos >= len(l.input) || !unicode.IsDigit(l.peek()) {
			return token{}, fmt.Errorf("%w: expected digit after '.' at position %d", ErrSyntax, startPos)
		}
		for l.pos < len(l.input) && unicode.IsDigit(l.peek()) {
			sb.WriteRune(l.advance())
		}
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
	t := p.peek()

	switch t.typ {
	case tokString:
		p.advance()
		return t.str, nil
	case tokInt:
		p.advance()
		n, err := strconv.Atoi(t.str)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid integer %q", ErrSyntax, t.str)
		}
		return n, nil
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
		return p.parseListOrDomain()
	default:
		return nil, fmt.Errorf("%w: unexpected token in value position at %d", ErrSyntax, t.pos)
	}
}

// parseListOrDomain disambiguates between a nested domain [(...), ...] and a plain list [1,2,3].
// If the first element after [ is '(' or a connector string, it tries domain parsing first.
func (p *parser) parseListOrDomain() (any, error) {
	next := p.peekAt(1) // token after '['

	// If first element looks like a domain (tuple start or connector), try domain parse.
	if next.typ == tokLParen || (next.typ == tokString && (next.str == "&" || next.str == "|" || next.str == "!")) {
		savePos := p.pos
		domain, err := p.parseDomain()
		if err == nil {
			return domain, nil
		}
		p.pos = savePos // backtrack on failure
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
	pos := 0
	for pos < len(terms) {
		count, err := validateAt(terms, pos)
		if err != nil {
			return []any{}, err
		}
		pos += count
	}
	return terms, nil
}

// validateAt validates the expression rooted at terms[pos] and returns
// the number of tokens consumed by that expression.
func validateAt(terms []any, pos int) (int, error) {
	if pos >= len(terms) {
		return 0, fmt.Errorf("%w: unexpected end of domain", ErrSyntax)
	}

	switch terms[pos] {
	case "&", "|":
		// Binary operator: consumes 1 (itself) + operand1 + operand2
		if pos+1 >= len(terms) {
			return 0, ErrNotEnoughAndOrTerms
		}
		count1, err := validateAt(terms, pos+1)
		if err != nil {
			return 0, err
		}
		if pos+1+count1 >= len(terms) {
			return 0, ErrNotEnoughAndOrTerms
		}
		count2, err := validateAt(terms, pos+1+count1)
		if err != nil {
			return 0, err
		}
		return 1 + count1 + count2, nil

	case "!":
		// Unary operator: consumes 1 (itself) + operand
		if pos+1 >= len(terms) {
			return 0, ErrNotEnoughNotTerms
		}
		count, err := validateAt(terms, pos+1)
		if err != nil {
			return 0, err
		}
		return 1 + count, nil

	default:
		// Leaf term (a []any tuple) — consumes 1
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
// Returns an empty slice for empty or trivially invalid input (no outer brackets).
// Returns ErrSyntax for structurally invalid domains that begin with '['.
// Returns ErrNotEnoughAndOrTerms / ErrNotEnoughNotTerms for prefix-notation arity violations.
func ParseDomain(domain string) (filter []any, err error) {
	domain = strings.TrimSpace(domain)
	if domain == "" {
		return []any{}, nil
	}

	// Tokenize
	tokens, lexErr := newLexer(domain).tokenize()
	if lexErr != nil {
		// Lex failure on input without brackets is not an error — just not a domain.
		return []any{}, nil
	}

	// Domain must start with [
	if len(tokens) == 0 || tokens[0].typ != tokLBracket {
		return []any{}, nil
	}

	// Parse
	p := newParser(tokens)
	result, parseErr := p.parseDomain()
	if parseErr != nil {
		if errors.Is(parseErr, ErrSyntax) {
			return []any{}, ErrSyntax
		}
		return []any{}, nil
	}

	// Must have consumed all tokens (except trailing EOF)
	if p.peek().typ != tokEOF {
		return []any{}, nil
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

// Fields splits a comma-separated field list into individual field names.
func Fields(field string) []string {
	if field != "" {
		return strings.Split(field, ",")
	}
	return []string{}
}
