package examples

import (
	"context"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/g"

	time "time"

	models "gorm.io/cmd/gorm/examples/models"
)

func Query[T any](db *gorm.DB, opts ...g.Option) QueryInterface[T] {
	return QueryImpl[T]{
		Interface: g.G[T](db, opts...),
	}
}

type QueryInterface[T any] interface {
	g.ChainInterface[T]
	GetByID(ctx context.Context, id int) (T, error)
	FilterWithColumn(ctx context.Context, column string, value string) (T, error)
	QueryWith(ctx context.Context, user models.User) (T, error)
	Update(ctx context.Context, user models.User, id int) error
	FilterByNameAndAge(ctx context.Context, name string, age int) QueryInterface[T]
	FilterWithTime(ctx context.Context, start time.Time, end time.Time) ([]T, error)
}

type QueryImpl[T any] struct {
	g.Interface[T]
}

func (e QueryImpl[T]) GetByID(ctx context.Context, id int) (T, error) {
	var sb strings.Builder
	sb.WriteString("SELECT * FROM @@table WHERE id=@id ")

	placeholderMap := map[string]any{
		"@table": clause.CurrentTable,
		"id":     id,
	}

	var result T
	err := e.Raw(sb.String(), placeholderMap).Scan(ctx, &result)
	return result, err
}

func (e QueryImpl[T]) FilterWithColumn(ctx context.Context, column string, value string) (T, error) {
	var sb strings.Builder
	sb.WriteString("SELECT * FROM @@table WHERE @@column=@value ")

	placeholderMap := map[string]any{
		"@table":  clause.CurrentTable,
		"@column": gorm.Expr("?", column),
		"value":   value,
	}

	var result T
	err := e.Raw(sb.String(), placeholderMap).Scan(ctx, &result)
	return result, err
}

func (e QueryImpl[T]) QueryWith(ctx context.Context, user models.User) (T, error) {
	var sb strings.Builder
	sb.WriteString("SELECT * FROM users ")
	if user.ID > 0 {
		sb.WriteString("WHERE id=@user.ID ")
	} else if user.Name != "" {
		sb.WriteString("WHERE username=@user.Name ")
	}

	placeholderMap := map[string]any{
		"user.ID":   user.ID,
		"user.Name": user.Name,
	}

	var result T
	err := e.Raw(sb.String(), placeholderMap).Scan(ctx, &result)
	return result, err
}

func (e QueryImpl[T]) Update(ctx context.Context, user models.User, id int) error {
	var sb strings.Builder
	sb.WriteString("UPDATE @@table ")
	{
		var tmp strings.Builder
		if user.Name != "" {
			tmp.WriteString("username=@user.Name, ")
		}
		if user.Age > 0 {
			tmp.WriteString("age=@user.Age, ")
		}
		if user.Age >= 18 {
			tmp.WriteString("is_adult=1 ")
		} else {
			tmp.WriteString("is_adult=0 ")
		}
		c := strings.TrimSpace(tmp.String())
		if c != "" {
			if strings.HasSuffix(c, ",") {
				c = strings.TrimRight(c, ",")
				c = strings.TrimSpace(c)
			}
			sb.WriteString("SET ")
			sb.WriteString(c)
		}
	}
	sb.WriteString("WHERE id=@id ")

	placeholderMap := map[string]any{
		"@table":    clause.CurrentTable,
		"user.Name": user.Name,
		"user.Age":  user.Age,
		"id":        id,
	}

	return e.Exec(ctx, sb.String(), placeholderMap)
}

func (e QueryImpl[T]) FilterByNameAndAge(ctx context.Context, name string, age int) QueryInterface[T] {
	var sb strings.Builder

	e.Where(sb.String())

	return e
}

func (e QueryImpl[T]) FilterWithTime(ctx context.Context, start time.Time, end time.Time) ([]T, error) {
	var sb strings.Builder
	sb.WriteString("SELECT * FROM @@table ")
	{
		var tmp strings.Builder
		if !start.IsZero() {
			tmp.WriteString("created_time > @start ")
		}
		if !end.IsZero() {
			tmp.WriteString("AND created_time < @end ")
		}
		c := strings.TrimSpace(tmp.String())
		if c != "" {
			sb.WriteString("WHERE ")
			sb.WriteString(c)
		}
	}

	placeholderMap := map[string]any{
		"@table": clause.CurrentTable,
		"start":  start,
		"end":    end,
	}

	var result []T
	err := e.Raw(sb.String(), placeholderMap).Scan(ctx, &result)
	return result, err
}
