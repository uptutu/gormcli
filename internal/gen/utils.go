package gen

import (
	"database/sql"
	"database/sql/driver"
	"go/ast"
	"go/token"
	"go/types"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/tools/go/packages"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

var (
	allowedInterfaces []types.Type
	namedTypesCache   = map[string]types.Type{}
)

func init() {
	var _ sql.Scanner
	var _ driver.Valuer
	var _ gorm.Valuer
	var _ schema.SerializerInterface

	// Load referenced interfaces from stdlib and gorm
	if scanner := loadNamedType("", "database/sql", "Scanner"); scanner != nil {
		allowedInterfaces = append(allowedInterfaces, scanner)
	}
	if valuer := loadNamedType("", "database/sql/driver", "Valuer"); valuer != nil {
		allowedInterfaces = append(allowedInterfaces, valuer)
	}
	if gormValuer := loadNamedType("", "gorm.io/gorm", "Valuer"); gormValuer != nil {
		allowedInterfaces = append(allowedInterfaces, gormValuer)
	}
	if serializer := loadNamedType("", "gorm.io/gorm/schema", "SerializerInterface"); serializer != nil {
		allowedInterfaces = append(allowedInterfaces, serializer)
	}
}

// ImplementsAllowedInterfaces reports whether typ or *typ implements any allowed interface.
func ImplementsAllowedInterfaces(typ types.Type) bool {
	if typ == nil {
		return false
	}
	if ptr, ok := typ.(*types.Pointer); ok {
		typ = ptr.Elem()
	}
	for _, t := range allowedInterfaces {
		iface, _ := t.Underlying().(*types.Interface)
		if iface == nil {
			continue
		}
		if types.Implements(typ, iface) || types.Implements(types.NewPointer(typ), iface) {
			return true
		}
	}
	return false
}

type ExtractedSQL struct {
	Raw    string
	Where  string
	Select string
}

func extractSQL(comment string, methodName string) ExtractedSQL {
	comment = strings.TrimSpace(comment)

	if index := strings.Index(comment, "\n\n"); index != -1 {
		if strings.Contains(comment[index+2:], methodName) {
			comment = comment[:index]
		} else {
			comment = comment[index+2:]
		}
	}

	sql := strings.TrimPrefix(comment, methodName)
	if strings.HasPrefix(sql, "where(") && strings.HasSuffix(sql, ")") {
		content := strings.TrimSuffix(strings.TrimPrefix(sql, "where("), ")")
		content = strings.Trim(content, "\"")
		content = strings.TrimSpace(content)
		return ExtractedSQL{Where: content}
	} else if strings.HasPrefix(sql, "select(") && strings.HasSuffix(sql, ")") {
		content := strings.TrimSuffix(strings.TrimPrefix(sql, "select("), ")")
		content = strings.Trim(content, "\"")
		content = strings.TrimSpace(content)
		return ExtractedSQL{Select: content}
	}
	return ExtractedSQL{Raw: sql}
}

func findGoModDir(filename string) string {
	cmd := exec.Command("go", "env", "GOMOD")
	cmd.Dir = filepath.Dir(filename)
	out, _ := cmd.Output()
	return filepath.Dir(string(out))
}

// getCurrentPackagePath gets the full import path of the current file's package
func getCurrentPackagePath(filename string) string {
	cfg := &packages.Config{
		Mode: packages.NeedName,
		Dir:  findGoModDir(filename),
	}

	pkgs, err := packages.Load(cfg, filepath.Dir(filename))
	if err == nil && len(pkgs) > 0 && pkgs[0].PkgPath != "" {
		return pkgs[0].PkgPath
	}

	// Fallback: derive import path from go.mod (module path + relative dir)
	modRoot := findGoModDir(filename)
	if modRoot == "" {
		return ""
	}
	data, rerr := os.ReadFile(filepath.Join(modRoot, "go.mod"))
	if rerr != nil {
		return ""
	}
	var modulePath string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			modulePath = strings.TrimSpace(strings.TrimPrefix(line, "module "))
			break
		}
	}
	if modulePath == "" {
		return ""
	}
	rel, rerr := filepath.Rel(modRoot, filepath.Dir(filename))
	if rerr != nil || rel == "." {
		return modulePath
	}
	return modulePath + "/" + filepath.ToSlash(rel)
}

// loadNamedType returns a named type from a package with basic caching.
func loadNamedType(modRoot, pkgPath, name string) types.Type {
	key := pkgPath + "." + name
	if t, ok := namedTypesCache[key]; ok {
		return t
	}
	cfg := &packages.Config{Mode: packages.NeedTypes | packages.NeedName, Dir: modRoot}
	pkgs, err := packages.Load(cfg, pkgPath)
	if err != nil || len(pkgs) == 0 || pkgs[0].Types == nil {
		return nil
	}
	if obj := pkgs[0].Types.Scope().Lookup(name); obj != nil {
		namedTypesCache[key] = obj.Type()
		return obj.Type()
	}
	return nil
}

// generateDBName generates database column name using GORM's NamingStrategy and COLUMN tag.
func generateDBName(fieldName, gormTag string) string {
	tagSettings := schema.ParseTagSetting(reflect.StructTag(gormTag).Get("gorm"), ";")
	if tagSettings["COLUMN"] != "" {
		return tagSettings["COLUMN"]
	}

	// Use GORM's NamingStrategy with IdentifierMaxLength: 64
	ns := schema.NamingStrategy{IdentifierMaxLength: 64}
	return ns.ColumnName("", fieldName)
}

// AllowedFieldByType returns true if the field type should be treated as a simple, allowed column type.
// Rules:
// - Primitive numbers, bool, string
// - time.Time, []byte
// - Any named type that implements one of the allowed interfaces
func AllowedFieldByType(expr ast.Expr, pkgAlias string, imports []Import, filePath string) bool {
	switch t := expr.(type) {
	case *ast.Ident:
		name := t.Name
		if name == "string" || name == "bool" || strings.Contains(name, "int") || strings.Contains(name, "float") {
			return true
		}
	case *ast.SelectorExpr:
		// time.Time
		if id, ok := t.X.(*ast.Ident); ok && id.Name == "time" && t.Sel.Name == "Time" {
			return true
		}
	case *ast.ArrayType:
		// []byte
		if id, ok := t.Elt.(*ast.Ident); ok && id.Name == "byte" {
			return true
		}
	case *ast.StarExpr:
		// Handle pointer by examining the element
		if AllowedFieldByType(t.X, pkgAlias, imports, filePath) {
			return true
		}
	}

	// Fallback to interface-based checks
	if typ := ResolveTypeFromExpr(expr, imports, filePath, pkgAlias); typ != nil {
		return ImplementsAllowedInterfaces(typ)
	}
	return false
}

// mergeImports appends imports from src into dst if not already present (by Path)
func mergeImports(dst *[]Import, src []Import) {
	existing := map[string]bool{}
	for _, i := range *dst {
		existing[i.Path] = true
	}
	for _, i := range src {
		if !existing[i.Path] {
			*dst = append(*dst, i)
			existing[i.Path] = true
		}
	}
}

// shouldSkipFile checks if a file contains the generated code header and should be skipped
func shouldSkipFile(filePath string) bool {
	if !strings.HasSuffix(filePath, ".go") {
		return true
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return false // If we can't read the file, don't skip it
	}

	// Convert to string and check for the generated code header
	fileContent := string(content)

	// Check for the exact generated code header
	return strings.Contains(fileContent, "// Code generated by 'gorm.io/cmd/gorm'. DO NOT EDIT.")
}

// ResolveTypeFromExpr attempts to resolve a types.Type for a field's AST expression.
// - Supports selector expressions (pkg.Type)
// - Supports identifiers from the current package (exported)
// - Ignores basic built-in types (lowercase idents)
// - Follows one level of pointers
func ResolveTypeFromExpr(expr ast.Expr, imports []Import, filePath string, preferAlias string) types.Type {
	aliasToPath := map[string]string{}
	for _, i := range imports {
		aliasToPath[i.Name] = i.Path
	}
	switch t := expr.(type) {
	case *ast.Ident:
		if len(t.Name) > 0 && unicode.IsLower(rune(t.Name[0])) {
			return nil
		}
		// Try preferred alias package first (used when resolving types from external ASTs)
		if preferAlias != "" {
			if pkgPath := aliasToPath[preferAlias]; pkgPath != "" {
				if tt := loadNamedType(findGoModDir(filePath), pkgPath, t.Name); tt != nil {
					return tt
				}
			}
		}
		// Exported ident in current package
		curr := getCurrentPackagePath(filePath)
		if curr == "" {
			return nil
		}
		return loadNamedType(findGoModDir(filePath), curr, t.Name)
	case *ast.SelectorExpr:
		if pkgIdent, ok := t.X.(*ast.Ident); ok {
			if pkgPath := aliasToPath[pkgIdent.Name]; pkgPath != "" {
				return loadNamedType(findGoModDir(filePath), pkgPath, t.Sel.Name)
			}
		}
	case *ast.StarExpr:
		return ResolveTypeFromExpr(t.X, imports, filePath, preferAlias)
	}
	return nil
}

// strLit returns the unquoted string if expr is a string literal; otherwise "".
func strLit(expr ast.Expr) string {
	if bl, ok := expr.(*ast.BasicLit); ok && bl.Kind == token.STRING {
		if s, err := strconv.Unquote(bl.Value); err == nil {
			return s
		}
	}

	return ""
}
