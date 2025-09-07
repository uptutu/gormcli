package genconfig

// Config is the per-package generator configuration.
//
// It can be declared in the source files that the generator scans, e.g.:
//
//	import (
//	    gencfg "gorm.io/cmd/gorm/genconfig"
//	    "gorm.io/cmd/gorm/field"
//	    "database/sql"
//	)
//
//	var _ = gencfg.Config{
//	    OutPath:      "examples/output",
//	    // Use typed instances so generator can capture import paths and methods.
//	    // Keys are source Go type instances; values are wrapper type instances.
//	    // e.g. sql.NullTime{} -> field.Time{}
//	    FieldTypeMap: map[any]any{sql.NullTime{}: field.Time{}},
//	    FieldNameMap: map[string]any{"last_login": field.Time{}},
//	}
//
// The generator will prioritize FieldNameMap over FieldTypeMap when deciding
// which wrapper type to use for a field.
type Config struct {
	// OutPath overrides the CLI output path for files in the same package
	// where this Config literal is found.
	OutPath string

	// FieldTypeMap maps a Go type instance (key) to a wrapper type instance (value).
	// Example: map[any]any{ sql.NullTime{}: field.Time{} }
	// The generator reads the AST to extract the type expressions from both
	// key and value, so it can infer import paths and render calls like
	// `field.Time{}.WithColumn(...)`.
	FieldTypeMap map[any]any

	// FieldNameMap maps a field or column name to a typed instance, same as
	// FieldTypeMap. Name matches check DB column first, then struct field name.
	FieldNameMap map[string]any

	FileLevel bool
}
