package field

import "gorm.io/gorm/clause"

// AssignerExpression combines a clause.Expression with an Assignments provider.
// Types implementing this can be used both in expressions (e.g., WHERE) and
// passed directly to Set(...), which accepts clause.Assigner in recent GORM versions.
type AssignerExpression interface {
	clause.Expression
	clause.Assigner
}

// colOpExpr is a generic expression wrapper that also provides Assignments
// so it can be passed directly to Set(...).
type colOpExpr struct {
	col  clause.Column
	sql  string
	vars []any
}

func (e colOpExpr) Build(builder clause.Builder) {
	clause.Expr{SQL: e.sql, Vars: e.vars}.Build(builder)
}

func (e colOpExpr) Assignments() []clause.Assignment {
	return []clause.Assignment{{Column: e.col, Value: e}}
}
