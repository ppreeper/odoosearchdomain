# odoosearchdomain

Go library for parsing and building Odoo search domain strings into `[]any` structures.

```go
import "github.com/ppreeper/odoosearchdomain"
```

## Parsing Domains

`ParseDomain` takes an Odoo domain string and returns a flat `[]any` slice where each element is either a term (`[]any{field, operator, value}`) or a logical connector (`string`).

```go
result, err := odoosearchdomain.ParseDomain(
    "[('name', '=', 'ABC'), '|', ('phone','ilike','7620'), ('mobile', 'ilike', '7620')]",
)
// result: []any{
//   []any{"name", "=", "ABC"},
//   "|",
//   []any{"phone", "ilike", "7620"},
//   []any{"mobile", "ilike", "7620"},
// }
```

### Supported value types

| Input syntax              | Go type  | Example                                      |
| ------------------------- | -------- | -------------------------------------------- |
| `'text'` or `"text"`      | `string` | `('name', '=', 'ABC')`                       |
| `123`, `-5`               | `int`    | `('ref', '=', 12345)`                        |
| `1.23`, `-0.5`            | `float64`| `('amount', '=', 123.45)`                    |
| `True`, `true`            | `bool`   | `('active', '=', True)`                      |
| `False`, `false`          | `bool`   | `('active', '=', False)`                     |
| `None`, `none`            | `nil`    | `('parent_id', '=', None)`                   |
| `[1, 2, 3]`              | `[]any`  | `('id', 'in', [1, 2, 3])`                    |
| `['a', 'b']`             | `[]any`  | `('month', 'in', ['April', 'May'])`          |
| `[('field','op',val)]`   | `[]any`  | `('line', 'any', [('qty', '<=', 0)])`        |

Both single quotes (`'...'`) and double quotes (`"..."`) are supported for strings.
Backslash escapes (`\'`, `\"`, `\\`) are handled within quoted strings.

### Errors

| Error                      | Meaning                                              |
| -------------------------- | ---------------------------------------------------- |
| `ErrSyntax`                | Invalid token, unknown operator, or malformed domain  |
| `ErrNotEnoughAndOrTerms`   | `&` or `\|` operator missing required 2 operands     |
| `ErrNotEnoughNotTerms`     | `!` operator missing required 1 operand              |

Input that lacks outer `[...]` brackets returns an empty slice with no error.

### Validation

`ValidateDomain` checks that a previously parsed `[]any` slice has correct prefix-notation arity for all logical operators. It is called automatically by `ParseDomain`, but is also available for direct use:

```go
terms := []any{"&", []any{"a", "=", "1"}, []any{"b", "=", "2"}}
validated, err := odoosearchdomain.ValidateDomain(terms)
```

## Building Domains

The `Domain` and `Term` types allow programmatic construction of domain structures.

```go
dom := odoosearchdomain.NewDomain()
dom.AddTerm("name", "=", "ABC")
dom.Or(
    odoosearchdomain.NewTerm("phone", "ilike", "7620"),
    odoosearchdomain.NewTerm("mobile", "ilike", "7620"),
)
list := dom.ToList()
// list: []any{
//   []any{"name", "=", "ABC"},
//   "|",
//   []any{"phone", "ilike", "7620"},
//   []any{"mobile", "ilike", "7620"},
// }
```

### Domain methods

| Method                          | Description                                          |
| ------------------------------- | ---------------------------------------------------- |
| `NewDomain() *Domain`           | Create an empty domain                               |
| `AddTerm(field, op, value)`     | Append a term                                        |
| `Add(term *Term)`               | Append a pre-built term                              |
| `And(term1, term2 *Term)`       | Append `"&"` followed by two terms                   |
| `Or(term1, term2 *Term)`        | Append `"\|"` followed by two terms                  |
| `Not(term *Term)`               | Append `"!"` followed by one term                    |
| `ToList() []any`                | Serialize to a flat `[]any` for JSON-RPC              |

### Utility functions

| Function                        | Description                                          |
| ------------------------------- | ---------------------------------------------------- |
| `DomainList(domains ...any)`    | Normalize variadic domain args into `[]any`          |
| `DomainString(domains ...string)` | Normalize variadic string args into `[]string`     |
| `Fields(field string)`          | Split a comma-separated field list                   |

## Odoo Search Domain Reference

A domain is a list of criteria, each criterion being a tuple of `(field_name, operator, value)` where:

### `field_name (str)`

A field name of the current model, or a relationship traversal through a Many2one using dot-notation e.g. `'street'` or `'partner_id.country'`. If the field is a date(time) field, you can also specify a part of the date using `'field_name.granularity'`. The supported granularities are `'year_number'`, `'quarter_number'`, `'month_number'`, `'iso_week_number'`, `'day_of_week'`, `'day_of_month'`, `'day_of_year'`, `'hour_number'`, `'minute_number'`, `'second_number'`. They all use an integer as value.

### `operator (str)`

An operator used to compare the field_name with the value. Valid operators are:

| Operator    | Description                                                                                                                                                                                                  |
| ----------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `=`         | equals to                                                                                                                                                                                                    |
| `!=`        | not equals to                                                                                                                                                                                                |
| `>`         | greater than                                                                                                                                                                                                 |
| `>=`        | greater than or equal to                                                                                                                                                                                     |
| `<`         | less than                                                                                                                                                                                                    |
| `<=`        | less than or equal to                                                                                                                                                                                        |
| `=?`        | unset or equals to (returns true if `value` is either `None` or `False`, otherwise behaves like `=`)                                                                                                         |
| `=like`     | matches `field_name` against the value pattern. An underscore `_` in the pattern stands for (matches) any single character; a percent sign `%` matches any string of zero or more characters.                |
| `like`      | matches `field_name` against the `%value%` pattern. Similar to `=like` but wraps value with `%` before matching                                                                                              |
| `not like`  | doesn't match against the `%value%` pattern                                                                                                                                                                  |
| `ilike`     | case insensitive `like`                                                                                                                                                                                      |
| `not ilike` | case insensitive `not like`                                                                                                                                                                                  |
| `=ilike`    | case insensitive `=like`                                                                                                                                                                                     |
| `in`        | is equal to any of the items from `value`, value should be a list of items                                                                                                                                   |
| `not in`    | is unequal to all of the items from `value`                                                                                                                                                                  |
| `child_of`  | is a child (descendant) of a `value` record (value can be either one item or a list of items). Takes the semantics of the model into account (i.e following the relationship field named by `_parent_name`). |
| `parent_of` | is a parent (ascendant) of a `value` record (value can be either one item or a list of items). Takes the semantics of the model into account (i.e following the relationship field named by `_parent_name`). |
| `any`       | matches if any record in the relationship traversal through `field_name` satisfies the provided domain value.                                                                                                |
| `not any`   | matches if no record in the relationship traversal through `field_name` satisfies the provided domain value.                                                                                                 |

### `value`

Variable type, must be comparable (through `operator`) to the named field.

### Logical operators

Domain criteria can be combined using logical operators in prefix (Polish) notation:

| Operator | Arity | Description                                        |
| -------- | ----- | -------------------------------------------------- |
| `'&'`    | 2     | Logical AND (default between consecutive criteria) |
| `'\|'`   | 2     | Logical OR                                         |
| `'!'`    | 1     | Logical NOT                                        |

Mostly to negate combinations of criteria. Individual criterion generally have a negative form (e.g. `=` -> `!=`, `<` -> `>=`) which is simpler than negating the positive.

### Examples

Search for partners named _ABC_, with a phone or mobile number containing _7620_:

```python
[('name', '=', 'ABC'), '|', ('phone','ilike','7620'), ('mobile', 'ilike', '7620')]
```

Search sales orders to invoice with at least one line where the product is out of stock:

```python
[('invoice_status', '=', 'to invoice'), ('order_line', 'any', [('product_id.qty_available', '<=', 0)])]
```

Search for all partners born in February:

```python
[('birthday.month_number', '=', 2)]
```

Nested AND with three terms:

```python
['&', '&', ('a', '=', '1'), ('b', '=', '2'), ('c', '=', '3')]
```

Negate an OR combination:

```python
['!', '|', ('active', '=', False), ('state', '=', 'draft')]
```

## Parser Architecture

The parser uses a two-phase approach:

**Phase 1 -- Lexer.** A single-pass O(n) tokenizer converts the input string into a stream of typed tokens (brackets, parens, commas, quoted strings, numbers, booleans, None).

**Phase 2 -- Recursive descent parser.** Consumes the token stream according to this grammar:

```
domain  -> '[' items ']' | '[]' | '[()]'
items   -> item (',' item)*
item    -> connector | term
connector -> STRING  {where str in '&', '|', '!'}
term    -> '(' STRING ',' STRING ',' value ')'
value   -> STRING | INT | FLOAT | TRUE | FALSE | NONE | list | domain
list    -> '[' (value (',' value)*)? ']'
```

Nested domains in value position (for `any`/`not any` operators) are handled by peek-based disambiguation with backtracking: if the first element after `[` is `(` or a connector string, domain parsing is attempted first.

After parsing, a recursive prefix-notation validator checks that `&`/`|` have exactly 2 operands and `!` has exactly 1 operand.
