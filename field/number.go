package field

import (
	"golang.org/x/exp/constraints"
	"gorm.io/gorm/clause"
)

type Number[T constraints.Integer | constraints.Float] struct {
	column clause.Column
}

func (n Number[T]) Eq(value T) clause.Expression {
	return clause.Eq{Column: n.column, Value: value}
}

func (n Number[T]) Neq(value T) clause.Expression {
	return clause.Neq{Column: n.column, Value: value}
}

func (n Number[T]) Gt(value T) clause.Expression {
	return clause.Gt{Column: n.column, Value: value}
}

func (n Number[T]) Gte(value T) clause.Expression {
	return clause.Gte{Column: n.column, Value: value}
}

func (n Number[T]) Lt(value T) clause.Expression {
	return clause.Lt{Column: n.column, Value: value}
}

func (n Number[T]) Lte(value T) clause.Expression {
	return clause.Lte{Column: n.column, Value: value}
}

func (n Number[T]) Between(v1, v2 T) clause.Expression {
	return clause.And(
		clause.Gte{Column: n.column, Value: v1},
		clause.Lte{Column: n.column, Value: v2},
	)
}

func (n Number[T]) In(values ...T) clause.Expression {
	interfaceValues := make([]any, len(values))
	for i, v := range values {
		interfaceValues[i] = v
	}
	return clause.IN{Column: n.column, Values: interfaceValues}
}

func (n Number[T]) NotIn(values ...T) clause.Expression {
	interfaceValues := make([]any, len(values))
	for i, v := range values {
		interfaceValues[i] = v
	}
	return clause.Not(clause.IN{Column: n.column, Values: interfaceValues})
}

func (n Number[T]) IsNull() clause.Expression {
	return clause.Expr{SQL: "? IS NULL", Vars: []any{n.column}}
}

func (n Number[T]) IsNotNull() clause.Expression {
	return clause.Expr{SQL: "? IS NOT NULL", Vars: []any{n.column}}
}
