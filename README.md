# GORM CMD

GORM CMD is a powerful command-line tool that simplifies database operations and generates type-safe query code for Go applications using GORM

✨ Features:
- Type-safe queries powered by Go generics
- SQL-like template DSL for flexible query definitions
- One command to generate clean, reusable code
- Fully integrated with GORM APIs

## Install

To use GORM CMD, ensure you have Go installed on your system (minimum version: Go 1.18+).

Run the following command to install:

```bash
go install gorm.io/cmd/gorm@latest
```

## Define Your Query Interface

You can define query interfaces in Go using generics and SQL-like templates.
GORM CMD will parse the interface and generate the corresponding query implementation.

```go
// ./examples/example.go
type Query[T any] interface {
  // GetByID queries data by ID and returns it as a struct.
  //
  // SELECT * FROM @@table WHERE id=@id
  GetByID(id int) (T, error)

  // SELECT * FROM @@table WHERE @@column=@value
  FilterWithColumn(column string, value string) (T, error)

  // SELECT * FROM users
  //   {{if user.ID > 0}}
  //       WHERE id=@user.ID
  //   {{else if user.Name != ""}}
  //       WHERE username=@user.Name
  //   {{end}}
  QueryWith(user models.User) (T, error)

  // UPDATE @@table
  //  {{set}}
  //    {{if user.Name != ""}} username=@user.Name, {{end}}
  //    {{if user.Age > 0}} age=@user.Age, {{end}}
  //    {{if user.Age >= 18}} is_adult=1 {{else}} is_adult=0 {{end}}
  //  {{end}}
  // WHERE id=@id
  Update(user models.User, id int) error

  // SELECT * FROM @@table
  // {{where}}
  //   {{for _, user := range users}}
  //     {{if user.Name != "" && user.Age > 0}}
  //       (username = @user.Name AND age=@user.Age AND role LIKE concat("%",@user.Role,"%")) OR
  //     {{end}}
  //   {{end}}
  // {{end}}
  Filter(users []models.User) ([]T, error)

  // where("name=@name AND age=@age")
  FilterByNameAndAge(name string, age int)

  // SELECT * FROM @@table
  //  {{where}}
  //    {{if !start.IsZero()}}
  //      created_time > @start
  //    {{end}}
  //    {{if !end.IsZero()}}
  //      AND created_time < @end
  //    {{end}}
  //  {{end}}
  FilterWithTime(start, end time.Time) ([]T, error)
}
```

## Generate Code

Run the generator with the following command:

```bash
gorm gen -i ./examples/example.go -o query
```

## Usage Examples

Import the generated query package and use it directly in your application:

```go
import "your_project/query"

// Query by ID
company, err := query.Query[Company].GetByID(ctx, 10)
// SELECT * FROM `companies` WHERE id=10

user, err := query.Query[User].GetByID(ctx, 10)
// SELECT * FROM `users` WHERE id=10

// Combine with other Generic APIs
err := query.Query[User].FilterByNameAndAge("jinzhu", 18).Delete(ctx)
// DELETE FROM `users` WHERE name='jinzhu' AND age=18

users, err := query.Query[User].FilterByNameAndAge("jinzhu", 18).Find(ctx)
// SELECT * FROM `users` WHERE name='jinzhu' AND age=18
```

## Template DSL Reference

GORM CMD uses a lightweight DSL to describe SQL-like templates.
Here are the key directives you can use:

| Directive    | Description                                   | Example                                       |
| ------------ | --------------------------------------------- | --------------------------------------------- |
| `@@table`    | Expands to the current model’s table name     | `SELECT * FROM @@table WHERE id=@id`          |
| `@@column`   | Expands to a column name passed as parameter  | `SELECT * FROM @@table WHERE @@column=@value` |
| `@param`     | Binds Go parameters into query                | `WHERE username=@user.Name`                   |
| `where`      | Starts a `WHERE` block, auto handles `AND/OR` | see example above                             |
| `set`        | Starts an `UPDATE SET` block                  | see example above                             |
| `if` / `for` | Control structures for dynamic SQL            | see example above                             |
