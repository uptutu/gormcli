# GORM CMD

GORM CMD is a code generation tool for Go applications that creates type-safe database query interfaces and field accessor methods for GORM. It generates compile-time verified database operations, eliminating runtime query errors.

## Features

- **Type-safe Queries**: Compile-time verification for database operations
- **SQL Templates**: Generate code from SQL template comments
- **Field Helpers**: Auto-generated type-safe field accessor methods
- **GORM Integration**: Works seamlessly with existing GORM APIs

## Installation

Requirements: Go 1.18 or later with generics support.

```bash
go install gorm.io/cmd/gorm@latest
```

## Quick Start

1. **Generate code from interface definitions:**
```bash
gorm gen -i ./query.go -o ./generated
```

2. **Use generated type-safe queries:**
```go
// Template-based queries
user, err := generated.UserQuery[User](db).GetByID(ctx, 1)

// Field-based queries
db.Where(generated.User.ID.Eq(1)).Find(&users)
```

## API Reference

### Template-based Query Generation

Template-based generation converts SQL comments into type-safe Go interface implementations with parameter binding and query optimization.

#### Basic Template Syntax

SQL template comments are embedded as Go interface method comments:

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

#### Template Interface Implementation

```go
import "your_project/generated"

user, err := generated.Query[User](db).GetByID(ctx, 123)
users, err := generated.Query[User](db).FilterByNameAndAge("jinzhu", 25).Find(ctx)
users, err := generated.Query[User](db).SearchUsers(ctx, User{Name: "jinzhu", Age: 25})
err := generated.Query[User](db).UpdateUser(ctx, updatedUser, 123)
```

### Struct Field Helper Generation

Field helper generation produces type-safe query builder methods for struct fields, enabling compile-time validation of database operations.

#### Model Definition

```go
type User struct {
    ID       uint
    Name     string
    Email    string
    Age      int
    Status   string
    CreateAt time.Time
}
```

#### Generated Field Operations

```go
// Equality operations
generated.User.ID.Eq(1)             // id = 1
generated.User.ID.Neq(1)            // id != 1
generated.User.ID.In(1, 2, 3)       // id IN (1, 2, 3)

// String operations
generated.User.Name.Eq("jinzhu")      // name = 'jinzhu'
generated.User.Name.Like("%jinzhu%")  // name LIKE '%jinzhu%'
generated.User.Name.IsNotNull()     // name IS NOT NULL

// Numeric operations
generated.User.Age.Gt(18)           // age > 18
generated.User.Age.Gte(18)          // age >= 18
generated.User.Age.Between(18, 65)  // age BETWEEN 18 AND 65
//... more checkout https://pkg.go.dev/gorm.io/cmd/gorm/field
```

#### Field Helper Usage

```go
import "your_project/generated"

// Basic queries
gorm.G[User](db).Where(generated.User.Status.Eq("active")).Find(ctx)
gorm.G[User](db).Where(generated.User.Age.Gte(18)).Find(ctx)

// Compound conditions
gorm.G[User](db).Where(
    generated.User.Age.Gt(18),
    generated.User.Status.Eq("active"),
).Find(&users)

// Data modification
gorm.G[User](db).Where(generated.User.Status.Eq("pending")).Update("status", "active")
```


## Template DSL Specification

GORM CMD implements a domain-specific language for SQL template generation with the following directives:

| Directive | Function | Implementation Example |
|-----------|----------|----------------------|
| `@@table` | Current model table name resolution | `SELECT * FROM @@table WHERE id=@id` |
| `@@column` | Dynamic column name parameter binding | `SELECT * FROM @@table WHERE @@column=@value` |
| `@param` | Go parameter to SQL parameter binding | `WHERE name=@user.Name` |
| `{{where}}` | Conditional WHERE clause generation | `{{where}} condition1 AND condition2 {{end}}` |
| `{{set}}` | Conditional SET clause generation for UPDATE | `{{set}} column1=@value1, column2=@value2 {{end}}` |
| `{{if}}` | Conditional SQL block generation | `{{if age > 0}} AND age=@age {{end}}` |
| `{{for}}` | Iterative SQL generation over collections | `{{for _, item := range items}} {{end}}` |

### Template Implementation Examples

```sql
-- Parameter binding with type safety
SELECT * FROM @@table WHERE id=@id AND status=@status

-- Conditional WHERE clause generation
SELECT * FROM @@table
{{where}}
  {{if name != ""}} name=@name {{end}}
  {{if age > 0}} AND age=@age {{end}}
{{end}}

-- Dynamic UPDATE with conditional field setting
UPDATE @@table
{{set}}
  {{if user.Name != ""}} name=@user.Name, {{end}}
  {{if user.Email != ""}} email=@user.Email {{end}}
{{end}}
WHERE id=@id

-- Iterative condition generation
SELECT * FROM @@table
{{where}}
  {{for _, tag := range tags}}
    {{if tag != ""}} tags LIKE concat('%',@tag,'%') OR {{end}}
  {{end}}
{{end}}
```
