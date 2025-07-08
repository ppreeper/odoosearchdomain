# odoosearchdomain

Odoo Search Domain Parser

From Odoo Documentation

## Search domains

A domain is a list of criteria, each criterion being a triple (either a list or a tuple) of (field_name, operator, value) where:

### `field_name (str)`

a field name of the current model, or a relationship traversal through a Many2one using dot-notation e.g. `'street'` or `'partner_id.country'`. If the field is a date(time) field, you can also specify a part of the date using `'field_name.granularity'`. The supported granularities are `'year_number'`, `'quarter_number'`, `'month_number'`, `'iso_week_number'`, `'day_of_week'`, `'day_of_month'`, `'day_of_year'`, `'hour_number'`, `'minute_number'`, `'second_number'`. They all use an integer as value.

### `operator (str)`

an operator used to compare the field_name with the value. Valid operators are:

| Operator    | Description                                                                                                                                                                                                  |
| ----------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `=`         | equals to                                                                                                                                                                                                    |
| `!=`        | not equals to                                                                                                                                                                                                |
| `>`         | greater than                                                                                                                                                                                                 |
| `>=`        | greater than or equal to                                                                                                                                                                                     |
| `<`         | less than                                                                                                                                                                                                    |
| `<=`        | less than or equal to                                                                                                                                                                                        |
| `=?`        | unset or equals to (returns true if `value` is either `None` or F`alse, otherwise behaves like `=`)                                                                                                          |
| `=like`     | matches `field_name` against the value pattern. An underscore `_` in the pattern stands for (matches) any single character; a percent sign `%` matches any string of zero or more characters.                |
| `like`      | matches `field_name` against the `%value%` pattern. Similar to `=like` but wraps value with `%` before matching                                                                                              |
| `not like`  | doesn’t match against the `%value%` pattern                                                                                                                                                                  |
| `ilike`     | case insensitive `like`                                                                                                                                                                                      |
| `not ilike` | case insensitive `not like`                                                                                                                                                                                  |
| `=ilike`    | case insensitive `=like`                                                                                                                                                                                     |
| `in`        | is equal to any of the items from `value`, value should be a list of items                                                                                                                                   |
| `not in`    | is unequal to all of the items from `value`                                                                                                                                                                  |
| `child_of`  | is a child (descendant) of a `value` record (value can be either one item or a list of items). Takes the semantics of the model into account (i.e following the relationship field named by `_parent_name`). |
| `parent_of` | is a parent (ascendant) of a `value` record (value can be either one item or a list of items). Takes the semantics of the model into account (i.e following the relationship field named by `_parent_name`). |
| `any`       | matches if any record in the relationship traversal through `field_name` (`Many2one`, `One2many`, or `Many2many`) satisfies the provided domain value.                                                       |
| `not any`   | matches if no record in the relationship traversal through `field_name` (`Many2one`, `One2many`, or `Many2many`) satisfies the provided domain value.                                                        |

### `value`

variable type, must be comparable (through `operator`) to the named field.

Domain criteria can be combined using logical operators in _prefix_ form:

`'&'`
logical _AND_, default operation to combine criteria following one another. Arity 2 (uses the next 2 criteria or combinations).

`'|'`
logical _OR_, arity 2.

`'!'`
logical NOT, arity 1.

Note: Mostly to negate combinations of criteria Individual criterion generally have a negative form (e.g. `=` -> `!=`, `<` -> `>=`) which is simpler than negating the positive.

Example

To search for partners named _ABC_, with a phone or mobile number containing _7620_:

```
[('name', '=', 'ABC'), '|', ('phone','ilike','7620'), ('mobile', 'ilike', '7620')]
```

To search sales orders to invoice that have at least one line with a product that is out of stock:

```
[('invoice_status', '=', 'to invoice'), ('order_line', 'any', [('product_id.qty_available', '<=', 0)])]
```

To search for all partners born in the month of February:

```
[('birthday.month_number', '=', 2)]
```


[(][^()]+?[)]|\||&|!