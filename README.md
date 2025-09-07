# GORM CMD

**GORM CMD** is a code generation tool for Go that produces **type-safe query interfaces** and **field helper methods** for GORM.
It eliminates runtime query errors by verifying database operations at **compile time**.

## 🚀 Features

* **Type-safe Queries** – Compile-time validation of database operations
* **SQL Templates** – Generate query methods directly from SQL template comments
* **Field Helpers** – Auto-generated, strongly typed field accessor methods
* **Seamless GORM Integration** – Works with existing GORM APIs out of the box

## 📦 Installation

Requires Go **1.18+** (with generics).

```bash
go install gorm.io/cmd/gorm@latest
```

## ⚡ Quick Start

### 1. Generate code from interfaces

```bash
gorm gen -i ./query.go -o ./generated
```

### 2. Run type-safe queries

```go
import "your_project/generated"

func Example(db *gorm.DB, ctx context.Context) {
    // Template-based query (from interface)
    user, err := generated.Query[User](db).GetByID(ctx, 123)

    // Field-based query (using generated helpers)
    users, err := gorm.G[User](db).
        Where(generated.User.Age.Gt(18)).
        Find(ctx)

    fmt.Println(user, users)
}
```

---


## 🔎 API Overview

### Template-based Query Generation

Define SQL templates in Go interfaces.
GORM CMD generates strongly typed implementations with parameter binding and compile-time validation.

```go
type Query[T any] interface {
    // SELECT * FROM @@table WHERE id=@id
    GetByID(id int) (T, error)

    // SELECT * FROM @@table WHERE @@column=@value
    FilterWithColumn(column string, value string) (T, error)

    // where("name=@name AND age=@age")
    FilterByNameAndAge(name string, age int)

    // SELECT * FROM @@table
    // {{where}}
    //   {{if @user.Name }} name=@user.Name {{end}}
    //   {{if @user.Age > 0}} AND age=@user.Age {{end}}
    // {{end}}
    SearchUsers(user User) ([]T, error)

    // UPDATE @@table
    // {{set}}
    //   {{if user.Name != ""}} name=@user.Name, {{end}}
    //   {{if user.Age > 0}} age=@user.Age {{end}}
    // {{end}}
    // WHERE id=@id
    UpdateUser(user User, id int) error
}
```

#### Usage

```go
import "your_project/generated"

func ExampleQuery(db *gorm.DB, ctx context.Context) {
    // Get a single user by ID
    user, err := generated.Query[User](db).GetByID(ctx, 123)

    // Filter users by dynamic column and value
    user, err := generated.Query[User](db).FilterWithColumn(ctx, "role", "admin")

    // Filter users by name and age
    users, err := generated.Query[User](db).FilterByNameAndAge("jinzhu", 25).Find(ctx)

    // Conditional search using template logic
    users, err := generated.Query[User](db).
        SearchUsers(ctx, User{Name: "jinzhu", Age: 25})

    // Update user with dynamic SET clause
    err := generated.Query[User](db).
        UpdateUser(ctx, updatedUser, 123)
}
```

---

### Field Helper Generation

Generate strongly typed field helpers for struct fields.
These enable expressive, compile-time validated queries.

#### Example Model

```go
type User struct {
    ID        uint
    Name      string
    Email     string
    Age       int
    Status    string
    CreatedAt time.Time
}
```

#### Generated Helpers

```go
// Equality
generated.User.ID.Eq(1)          // id = 1
generated.User.ID.Neq(1)         // id != 1
generated.User.ID.In(1, 2, 3)    // id IN (1, 2, 3)

// String
generated.User.Name.Like("%jinzhu%") // name LIKE '%jinzhu%'
generated.User.Name.IsNotNull()      // name IS NOT NULL

// Numeric
generated.User.Age.Gt(18)            // age > 18
generated.User.Age.Between(18, 65)   // age BETWEEN 18 AND 65

// Nullable (Scanner/Valuer) types
generated.User.Score.IsNull()        // score IS NULL (sql.NullInt64)
generated.User.LastLogin.IsNotNull() // last_login IS NOT NULL (sql.NullTime)

// ... more, see https://pkg.go.dev/gorm.io/cmd/gorm/field
```

#### Usage

```go
// Simple filter
gorm.G[User](db).
    Where(generated.User.Status.Eq("active")).
    Find(ctx)

// Multiple conditions
gorm.G[User](db).
    Where(generated.User.Age.Gt(18), generated.User.Status.Eq("active")).
    Find(&users)

// Update using helpers
gorm.G[User](db).
    Where(generated.User.Status.Eq("pending")).
    Update("status", "active")
```

---

## 🧠 How Fields Are Chosen

GORM CMD generates field helpers only for “column-like” fields on your model and skips associations.

- Included:
  - Built-ins: all integers, floats, `string`, `bool`, `time.Time`, `[]byte`
  - Any named type implementing one of:
    - `database/sql.Scanner`
    - `database/sql/driver.Valuer`
    - `gorm.io/gorm.Valuer`
    - `gorm.io/gorm/schema.SerializerInterface`
- Excluded:
  - Associations such as `has one`, `has many`, `belongs to`, `many2many`, and embedded slices/structs used as relations

---

## 📝 Template DSL

GORM CMD provides a SQL template DSL:

| Directive   | Purpose                            | Example                                  |
| ----------- | ---------------------------------- | ---------------------------------------- |
| `@@table`   | Resolves to the model’s table name | `SELECT * FROM @@table WHERE id=@id`     |
| `@@column`  | Dynamic column binding             | `@@column=@value`                        |
| `@param`    | Maps Go params to SQL params       | `WHERE name=@user.Name`                  |
| `{{where}}` | Conditional WHERE clause           | `{{where}} age > 18 {{end}}`             |
| `{{set}}`   | Conditional SET clause (UPDATE)    | `{{set}} name=@name {{end}}`             |
| `{{if}}`    | Conditional SQL fragment           | `{{if age > 0}} AND age=@age {{end}}`    |
| `{{for}}`   | Iteration over a collection        | `{{for _, t := range tags}} ... {{end}}` |

### Examples

```sql
-- Safe parameter binding
SELECT * FROM @@table WHERE id=@id AND status=@status

-- Dynamic column binding
SELECT * FROM @@table WHERE @@column=@value

-- Conditional WHERE
SELECT * FROM @@table
{{where}}
  {{if name != ""}} name=@name {{end}}
  {{if age > 0}} AND age=@age {{end}}
{{end}}

-- Dynamic UPDATE
UPDATE @@table
{{set}}
  {{if user.Name != ""}} name=@user.Name, {{end}}
  {{if user.Email != ""}} email=@user.Email {{end}}
{{end}}
WHERE id=@id

-- Iteration
SELECT * FROM @@table
{{where}}
  {{for _, tag := range tags}}
    {{if tag != ""}} tags LIKE concat('%',@tag,'%') OR {{end}}
  {{end}}
{{end}}
```

---

## Generation Config

Declare a package-level `genconfig.Config` in the package you want to generate for. The generator will pick it up automatically:

```go
package examples

import (
    "database/sql"
    "gorm.io/cmd/gorm/field"
    "gorm.io/cmd/gorm/genconfig"
)

var _ = genconfig.Config{
    // Output directory for generated files in this package (overrides CLI -o)
    OutPath: "examples/output",

    // Map Go types to field helper types
    FieldTypeMap: map[any]any{
        sql.NullTime{}: field.Time{},
    },

    FieldNameMap: map[string]any{
        "date": field.Time{},  // map fields with `gen:"date"` tag to Time field helper
        "json": JSON{},        // map fields with `gen:"json"` tag to custom JSON helper
    },

    // If true, the config applies only to the current file rather than the entire package
    FileLevel: false,
}
```

### JSON Field Mapping Example

0) Declare Configuration

```go
package examples

import "gorm.io/cmd/gorm/genconfig"

var _ = genconfig.Config{
    OutPath: "examples/output",
    FieldNameMap: map[string]any{
        "json": JSON{},        // map fields with `gen:"json"` tag to custom JSON helper
    },
}
```

1) Declare JSON on the model using struct tags

```go
package models

type User struct {
    // ... other fields ...
    // Tell the generator to use the custom JSON helper for this column
    Profile string `gen:"json"`
}
```

2) Define the JSON helper

```go
// JSON is a field helper for JSON columns that generates different SQL for different databases.
type JSON struct{ column clause.Column }

func (j JSON) WithColumn(name string) JSON {
    c := j.column
    c.Name = name
    return JSON{column: c}
}

// Equal builds an expression using database-specific JSON functions to compare
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
            v, _ := json.Marshal(e.val)
            clause.Expr{SQL: "JSON_EXTRACT(?, ?) = CAST(? AS JSON)", Vars: []any{e.col, e.path, string(v)}}.Build(builder)
        case "sqlite":
            clause.Expr{SQL: "json_valid(?) AND json_extract(?, ?) = ?", Vars: []any{e.col, e.col, e.path, e.val}}.Build(builder)
        default:
            clause.Expr{SQL: "jsonb_extract_path_text(?, ?) = ?", Vars: []any{e.col, e.path[2:], e.val}}.Build(builder)
        }
    }
}
```

3) Use it in queries

```go
// This will generate different SQL depending on the database:
// MySQL:  "JSON_EXTRACT(`profile`, "$.vip") = CAST("true" AS JSON)"
// SQLite: "json_valid(`profile`) AND json_extract(`profile`, "$.vip") = 1"
got, err := gorm.G[models.User](db).
    Where(generated.User.Profile.Equal("$.vip", true)).Take(ctx)
```
