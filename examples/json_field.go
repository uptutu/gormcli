package examples

import (
	"encoding/json"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

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
	return jsonEqualExpr{col: j.column, path: path, val: value}
}

type jsonEqualExpr struct {
	col  clause.Column
	path string
	val  any
}

func (e jsonEqualExpr) Build(builder clause.Builder) {
	if stmt, ok := builder.(*gorm.Statement); ok {
		switch stmt.Dialector.Name() {
		case "mysql":
			// Compare JSON to JSON using JSON_EXTRACT(column, path) = CAST(? AS JSON)
			// This avoids dialect boolean quirks and works for all JSON scalars and null.
			valJSON, _ := json.Marshal(e.val)
			builder.WriteString("JSON_EXTRACT(")
			builder.AddVar(builder, e.col)
			builder.WriteString(", ")
			builder.AddVar(builder, e.path)
			builder.WriteString(") = CAST(")
			builder.AddVar(builder, string(valJSON))
			builder.WriteString(" AS JSON)")
		case "sqlite":
			// SQLite: guard invalid JSON and compare scalar via json_extract
			builder.WriteString("json_valid(")
			builder.AddVar(builder, e.col)
			builder.WriteString(") AND json_extract(")
			builder.AddVar(builder, e.col)
			builder.WriteString(", ")
			builder.AddVar(builder, e.path)
			builder.WriteString(") = ")
			builder.AddVar(builder, e.val)
		default:
			builder.WriteString("JSON_EXTRACT(")
			builder.AddVar(builder, e.col)
			builder.WriteString(", ")
			builder.AddVar(builder, e.path)
			builder.WriteString(") = ")
			builder.AddVar(builder, e.val)
		}
	}
}
