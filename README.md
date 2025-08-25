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

1. **Generate code from interfaces:**

```bash
gorm gen -i ./query.go -o ./generated
```

2. **Use generated type-safe queries:**

```go
// Template-based query
user, err := generated.UserQuery[User](db).GetByID(ctx, 1)

// Field-based query
db.Where(generated.User.ID.Eq(1)).Find(&users)
```

## 🔎 API Overview

### Template-based Query Generation

Define SQL templates in Go interfaces. GORM CMD generates strongly typed implementations with parameter binding and compile-time validation.

```go
type Query[T any] interface {
    // SELECT * FROM @@table WHERE id=@id
    GetByID(id int) (T, error)

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

Usage:

```go
import "your_project/generated"

user, err := generated.Query[User](db).GetByID(ctx, 123)
users, err := generated.Query[User](db).FilterByNameAndAge("jinzhu", 25).Find(ctx)
users, err := generated.Query[User](db).SearchUsers(ctx, User{Name: "jinzhu", Age: 25})
err := generated.Query[User](db).UpdateUser(ctx, updatedUser, 123)
```

---

### Field Helper Generation

Generate strongly typed field helpers for struct fields. These enable expressive, compile-time validated queries.

#### Example Model

```go
type User struct {
    ID       uint
    Name     string
    Email    string
    Age      int
    Status   string
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

// ... more, see https://pkg.go.dev/gorm.io/cmd/gorm/field
```

#### Usage

```go
gorm.G[User](db).
    Where(generated.User.Status.Eq("active")).
    Find(ctx)

gorm.G[User](db).
    Where(generated.User.Age.Gt(18), generated.User.Status.Eq("active")).
    Find(&users)

gorm.G[User](db).
    Where(generated.User.Status.Eq("pending")).
    Update("status", "active")
```

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
