package examples

import "gorm.io/gorm/clause"

// JSON is an example field wrapper for JSON columns.
//
// It demonstrates how to create a custom field type with only one
// operation (Contains) and the required WithColumn method so it can be
// used in config mappings and query building.
//
// Note: The SQL generated here uses MySQL-style JSON_CONTAINS for
// demonstration purposes. Adapt the SQL if you target a different DB.
type JSON struct {
	column clause.Column
}

// WithColumn sets the column name for this JSON field.
func (j JSON) WithColumn(name string) JSON {
	c := j.column
	c.Name = name
	return JSON{column: c}
}

// Contains creates a JSON containment predicate.
// Example (MySQL): JSON_CONTAINS(column, @value)
func (j JSON) Contains(value any) clause.Expression {
	return clause.Expr{SQL: "JSON_CONTAINS(?, ?)", Vars: []any{j.column, value}}
}

// Equal builds an expression using SQLite's JSON1 extension to compare
// the JSON value at the given JSON path with the provided value.
// Example: json_extract(column, '$.vip') = 1
// Path must be a valid JSON path like "$.vip".
func (j JSON) Equal(path string, value any) clause.Expression {
	// Guard with json_valid to avoid "malformed JSON" errors for empty/non-JSON text
	// Uses column twice: once for json_valid, once for json_extract
	return clause.Expr{SQL: "json_valid(?) AND json_extract(?, ?) = ?", Vars: []any{j.column, j.column, path, value}}
}
