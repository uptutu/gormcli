package gen

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"unicode"

	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/imports"
	"gorm.io/cmd/gorm/genconfig"
)

type Generator struct {
	Files   map[string]*File
	outPath string
}

// Process processes input files or directories and generates code
func (g *Generator) Process(input string) error {
	info, err := os.Stat(input)
	if err != nil {
		return err
	}

	// Store the input root for relative path calculation
	if info.IsDir() {
		inputRoot, _ := filepath.Abs(input)
		return filepath.Walk(input, func(path string, info os.FileInfo, err error) error {
			if err == nil && !info.IsDir() {
				return g.processFile(path, inputRoot)
			}
			return err
		})
	}
	inputRoot, _ := filepath.Abs(filepath.Dir(input))
	return g.processFile(input, inputRoot)
}

// Gen generates code files from processed AST data
func (g *Generator) Gen() error {
	tmpl, err := template.New("").Parse(pkgTmpl)
	if err != nil {
		panic(err)
	}

	fileCfgs := []string{}
	for pth, file := range g.Files {
		if file.Config != nil {
			fileCfgs = append(fileCfgs, pth)
		}
	}
	sort.Strings(fileCfgs)

	for _, file := range g.Files {
		if len(file.Interfaces) == 0 && len(file.Structs) == 0 {
			continue
		}

		outPath := g.outPath
		for i := len(fileCfgs) - 1; i >= 0; i-- {
			prefix := fileCfgs[i]
			if !g.Files[fileCfgs[i]].Config.FileLevel {
				prefix = filepath.Dir(fileCfgs[i])
			}
			if strings.HasPrefix(file.inputPath, prefix) {
				if outPath == "" {
					outPath = g.Files[fileCfgs[i]].Config.OutPath
				}

				file.applicableConfigs = append(file.applicableConfigs, g.Files[fileCfgs[i]].Config)
				mergeImports(&file.Imports, g.Files[fileCfgs[i]].Imports)
			}
		}
		if outPath == "" {
			outPath = "./g"
		}
		outPath = filepath.Join(outPath, file.relPath)

		if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
			panic(fmt.Sprintf("failed to create directory for %v, got error %v", outPath, err))
		}

		f, err := os.Create(outPath)
		if err != nil {
			panic(fmt.Sprintf("failed to create file %v, got error %v", outPath, err))
		}

		fmt.Printf("Generating file %s from %s...\n", outPath, file.inputPath)
		if err := tmpl.Execute(f, file); err != nil {
			panic(fmt.Sprintf("failed to render template %v, got error %v", file.inputPath, err))
		}

		// Ensure file is closed before formatting pass reads it
		if err := f.Close(); err != nil {
			panic(fmt.Sprintf("failed to close file %v, got error %v", outPath, err))
		}

		if result, err := imports.Process(outPath, nil, nil); err == nil {
			if err := os.WriteFile(outPath, result, 0o640); err != nil {
				panic(fmt.Sprintf("failed to write file %v, got error %v", outPath, err))
			}
		}
	}
	return nil
}

// processFile processes a single Go file and extracts AST information
func (g *Generator) processFile(inputFile, inputRoot string) error {
	inputFile, err := filepath.Abs(inputFile)
	if err != nil {
		return err
	}

	if shouldSkipFile(inputFile) {
		fmt.Printf("Skipping generated file: %s\n", inputFile)
		return nil
	}

	// Calculate relative path from input root to maintain directory structure
	relPath, err := filepath.Rel(inputRoot, inputFile)
	if err != nil {
		relPath = filepath.Base(inputFile)
	}

	fileset := token.NewFileSet()
	f, err := parser.ParseFile(fileset, inputFile, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("can't parse file %q: %s", inputFile, err)
	}

	file := &File{Package: f.Name.Name, inputPath: inputFile, relPath: relPath}

	// Add current package to imports for alias/path resolution and generation needs
	if pkgPath := getCurrentPackagePath(inputFile); pkgPath != "" {
		file.Imports = append(file.Imports, Import{
			Name: f.Name.Name,
			Path: pkgPath,
		})
	}

	ast.Walk(file, f)

	// Store every processed file so configs in any file are discoverable
	g.Files[file.inputPath] = file

	if len(file.Interfaces) == 0 && len(file.Structs) == 0 {
		fmt.Printf("Skipping generated file: %s\n", inputFile)
	}
	return nil
}

type (
	File struct {
		Package           string
		Imports           []Import
		Interfaces        []Interface
		Structs           []Struct
		Config            *genconfig.Config
		applicableConfigs []*genconfig.Config
		inputPath         string
		relPath           string
	}
	Import struct {
		Name string
		Path string
	}
	Interface struct {
		Name      string
		IfaceName string
		Doc       string
		Methods   []*Method
	}
	Method struct {
		Name      string
		Doc       string
		SQL       ExtractedSQL
		Params    []Param
		Result    []Param
		Interface Interface
	}
	Param struct {
		Name string
		Type string
	}
	Struct struct {
		Name   string
		Doc    string
		Fields []Field
	}
	Field struct {
		Name        string
		DBName      string
		GoType      string
		GoTypeAlias string
		Tag         string
		file        *File
	}
)

// ImportPath returns formatted import path string for template generation
func (p Import) ImportPath() string {
	if path.Base(p.Path) == p.Name {
		return fmt.Sprintf("%q", p.Path)
	}
	return fmt.Sprintf("%s %q", p.Name, p.Path)
}

// GoFullType returns the complete Go type string for a parameter
func (p Param) GoFullType() string {
	return p.Type
}

// ParamsString formats method parameters as a string for code generation
func (m Method) ParamsString() string {
	var parts []string
	hasCtx := false

	for _, p := range m.Params {
		if p.Name == "ctx" || p.Type == "context.Context" {
			hasCtx = true
			p.Name = "ctx"
		}

		parts = append(parts, fmt.Sprintf("%s %s", p.Name, p.GoFullType()))
	}

	if !hasCtx {
		parts = append([]string{"ctx context.Context"}, parts...)
	}

	return strings.Join(parts, ", ")
}

// ResultString formats method return values as a string for code generation
func (m Method) ResultString() string {
	if m.SQL.Raw != "" {
		var rets []string
		for _, r := range m.Result {
			rets = append(rets, r.GoFullType())
		}

		return strings.Join(rets, ", ")
	}
	return fmt.Sprintf("%sInterface[T]", m.Interface.IfaceName)
}

// Body generates the method body code for templates
func (m Method) Body() string {
	if m.SQL.Raw != "" {
		return m.finishMethodBody()
	}
	return m.chainMethodBody()
}

// processSQL processes SQL template strings and returns formatted SQL snippet
func (m Method) processSQL(sql string) string {
	sqlSnippet, err := RenderSQLTemplate(sql)
	if err != nil {
		panic(fmt.Sprintf("Failed to parsing SQL template for %s.%s %q: %v", m.Interface.Name, m.Name, m.SQL, err))
	}

	return sqlSnippet
}

// finishMethodBody generates method body for finishing SQL operations that return data
func (m Method) finishMethodBody() string {
	sqlSnippet := m.processSQL(m.SQL.Raw)

	if len(m.Result) == 1 {
		return fmt.Sprintf(`%s
return e.Exec(ctx, sb.String(), params...)`, sqlSnippet)
	}

	return fmt.Sprintf(`%s
var result %s
err := e.Raw(sb.String(), params...).Scan(ctx, &result)
return result, err`, sqlSnippet, m.Result[0].GoFullType())
}

// chainMethodBody generates method body for chaining SQL operations that return interface
func (m Method) chainMethodBody() string {
	var callMethod, sql string
	if m.SQL.Select != "" {
		callMethod = "Select"
		sql = m.SQL.Select
	} else {
		callMethod = "Where"
		sql = m.SQL.Where
	}

	sqlSnippet := m.processSQL(sql)

	return fmt.Sprintf(`%s

e.%s(sb.String(), params...)

return e`, sqlSnippet, callMethod)
}

// parseFieldList converts AST field list to parameter slice for method signatures
func (p *File) parseFieldList(fields *ast.FieldList) []Param {
	if fields == nil {
		return nil
	}

	var params []Param
	for _, field := range fields.List {
		names := field.Names
		if len(names) == 0 {
			names = []*ast.Ident{{Name: ""}}
		}

		for _, n := range names {
			params = append(params, Param{
				Name: n.Name,
				Type: p.parseFieldType(field.Type, ""),
			})
		}
	}

	return params
}

var typeMap = map[string]string{
	"string":    "field.String",
	"bool":      "field.Bool",
	"[]byte":    "field.Bytes",
	"time.Time": "field.Time",
}

// Type returns the field type string for template generation
func (f Field) Type() string {
	goType := strings.TrimPrefix(f.GoType, "*")
	for _, cfg := range f.file.applicableConfigs {
		if v, ok := cfg.FieldNameMap[f.GoTypeAlias]; ok {
			return fmt.Sprint(v)
		}

		if v, ok := cfg.FieldTypeMap[f.GoType]; ok {
			return fmt.Sprint(v)
		}
	}

	if mapped, ok := typeMap[goType]; ok {
		return mapped
	}
	if strings.Contains(goType, "int") || strings.Contains(goType, "float") {
		return fmt.Sprintf("field.Number[%s]", goType)
	}
	return fmt.Sprintf("field.Field[%s]", goType)
}

// Value returns the field value string with column name for template generation
func (f Field) Value() string {
	return f.Type() + fmt.Sprintf("{}.WithColumn(%q)", f.DBName)
}

// Visit implements ast.Visitor to traverse AST nodes and extract imports, interfaces, and structs
func (p *File) Visit(n ast.Node) (w ast.Visitor) {
	switch n := n.(type) {
	case *ast.ImportSpec:
		importPath, _ := strconv.Unquote(n.Path.Value)
		importName := path.Base(importPath)
		if n.Name != nil {
			importName = n.Name.Name
		}

		p.Imports = append(p.Imports, Import{
			Name: importName,
			Path: importPath,
		})
	case *ast.GenDecl:
		if n.Tok == token.VAR {
			for _, spec := range n.Specs {
				if vs, ok := spec.(*ast.ValueSpec); ok {
					if cfg := p.tryParseConfig(vs); cfg != nil {
						p.Config = cfg
					}
				}
			}
		}
	case *ast.TypeSpec:
		if data, ok := n.Type.(*ast.InterfaceType); ok {
			p.Interfaces = append(p.Interfaces, p.processInterfaceType(n, data))
		} else if data, ok := n.Type.(*ast.StructType); ok {
			if s := p.processStructType(n, data, ""); len(s.Fields) > 0 {
				p.Structs = append(p.Structs, s)
			}
		}
	}
	return p
}

// tryParseConfig attempts to parse a gorm.io/cmd/gorm/genconfig.Config composite literal
// from a package-level value spec. Returns nil if not present.
func (p *File) tryParseConfig(vs *ast.ValueSpec) *genconfig.Config {
	// Helper to check whether a given expression is a selector to the imported Config type
	isCmdConfigType := func(expr ast.Expr) bool {
		sel, ok := expr.(*ast.SelectorExpr)
		if !ok || sel.Sel == nil || sel.Sel.Name != "Config" {
			return false
		}
		id, ok := sel.X.(*ast.Ident)
		if !ok {
			return false
		}
		// Find this ident's import path
		for _, i := range p.Imports {
			if i.Name == id.Name && i.Path == "gorm.io/cmd/gorm/genconfig" {
				return true
			}
		}
		return false
	}

	// Case 1: explicit type on the var
	if vs.Type != nil && isCmdConfigType(vs.Type) {
		for _, v := range vs.Values {
			if cl, ok := v.(*ast.CompositeLit); ok {
				if cfg := p.parseConfigLiteral(cl); cfg != nil {
					return cfg
				}
			}
		}
	}
	// Case 2: type is specified on the composite literal itself
	for _, v := range vs.Values {
		if cl, ok := v.(*ast.CompositeLit); ok && isCmdConfigType(cl.Type) {
			if cfg := p.parseConfigLiteral(cl); cfg != nil {
				return cfg
			}
		}
	}
	return nil
}

// parseConfigLiteral parses a cmd.Config composite literal into a Config value.
func (p *File) parseConfigLiteral(cl *ast.CompositeLit) *genconfig.Config {
	cfg := &genconfig.Config{FieldTypeMap: map[any]any{}, FieldNameMap: map[string]any{}}
	for _, elt := range cl.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		keyIdent, ok := kv.Key.(*ast.Ident)
		if !ok {
			continue
		}
		switch keyIdent.Name {
		case "OutPath":
			cfg.OutPath = strLit(kv.Value)
		case "FileLevel":
			if ident, ok := kv.Value.(*ast.Ident); ok {
				cfg.FileLevel = ident.Name == "true"
			}
		case "FieldTypeMap", "FieldNameMap":
			if m, ok := kv.Value.(*ast.CompositeLit); ok {
				for _, me := range m.Elts {
					if pair, ok := me.(*ast.KeyValueExpr); ok {
						// Values are wrapper type instances like JSON{} or field.Time{}
						// Use the current file's package to qualify local identifiers
						valType := p.parseFieldType(pair.Value, p.Package)
						if keyIdent.Name == "FieldNameMap" {
							// Keys are strings for FieldNameMap
							if key := strLit(pair.Key); key != "" && valType != "" {
								cfg.FieldNameMap[key] = valType
							}
						} else {
							// Keys are Go types for FieldTypeMap
							if key := p.parseFieldType(pair.Key, ""); key != "" && valType != "" {
								cfg.FieldTypeMap[key] = valType
							}
						}
					}
				}
			}
		}
	}
	return cfg
}

// processInterfaceType processes an interface type AST node and extracts interface metadata and methods
func (p *File) processInterfaceType(n *ast.TypeSpec, data *ast.InterfaceType) Interface {
	r := Interface{
		Name:      n.Name.Name,
		IfaceName: "_" + n.Name.Name,
		Doc:       n.Doc.Text(),
	}

	methods := data.Methods.List
	for _, m := range methods {
		for _, name := range m.Names {
			method := &Method{
				Name:      name.Name,
				Doc:       m.Doc.Text(),
				SQL:       extractSQL(m.Doc.Text(), name.Name),
				Interface: r,
			}
			r.Methods = append(r.Methods, method)

			method.Params = p.parseFieldList(m.Type.(*ast.FuncType).Params)
			method.Result = p.parseFieldList(m.Type.(*ast.FuncType).Results)

			if len(method.Result) == 0 {
				if method.SQL.Where == "" && method.SQL.Select == "" || method.SQL.Raw != "" {
					panic(fmt.Sprintf("Method %s.%s: finish method must return at least one value (last return value must be error)", n.Name.Name, method.Name))
				}
			} else if len(method.Result) > 2 {
				panic(fmt.Sprintf("Method %s.%s: maximum number of return values allowed is 2 (first as data, second as error)", n.Name.Name, method.Name))
			} else if strings.ToLower(method.Result[len(method.Result)-1].Type) != "error" {
				if len(method.Result) == 1 {
					panic(fmt.Sprintf("Method %s.%s: when only one return value is defined, its type must be error", n.Name.Name, method.Name))
				}
				panic(fmt.Sprintf("Method %s.%s: when two return values are defined, the second must be error", n.Name.Name, method.Name))
			}
		}
	}
	return r
}

// processStructType processes a struct type AST node and extracts struct metadata and fields
func (p *File) processStructType(typeSpec *ast.TypeSpec, data *ast.StructType, pkgName string) Struct {
	s := Struct{
		Name: typeSpec.Name.Name,
	}

	for _, field := range data.Fields.List {
		// Handle anonymous embedding first
		if len(field.Names) == 0 {
			if p.handleAnonymousEmbedding(field, pkgName, &s) {
				continue
			}
		}

		// Parse field type and names
		fieldType := p.parseFieldType(field.Type, pkgName)

		// Get field tag for DBName generation
		var fieldTag string
		if field.Tag != nil {
			fieldTag, _ = strconv.Unquote(field.Tag.Value)
		}

		// Only keep allowed fields; skip associations and unhandled complex types
		if !p.isAllowedField(field, pkgName) {
			continue
		}

		// Add fields to struct
		for _, n := range field.Names {
			if n.IsExported() {
				dbName := generateDBName(n.Name, fieldTag)
				f := Field{
					Name:        n.Name,
					DBName:      dbName,
					GoType:      fieldType,
					GoTypeAlias: reflect.StructTag(fieldTag).Get("gen"),
					Tag:         fieldTag,
					file:        p,
				}
				s.Fields = append(s.Fields, f)
			}
		}
	}

	return s
}

// isAssociationField determines whether a field should be treated as an association and skipped
// Keep primitives, time.Time, []byte, gorm.DeletedAt, and any type implementing
// one of: database/sql.Scanner, database/sql/driver.Valuer, gorm.Valuer, schema.SerializerInterface.
func (p *File) isAllowedField(field *ast.Field, pkgName string) bool {
	return AllowedFieldByType(field.Type, pkgName, p.Imports, p.inputPath)
}

// parseFieldType extracts the type string from an AST field type expression
func (p *File) parseFieldType(expr ast.Expr, pkgName string) string {
	switch t := expr.(type) {
	case *ast.Ident:
		// For basic Go types, don't add package prefix
		if len(t.Name) > 0 && unicode.IsLower(rune(t.Name[0])) {
			return t.Name
		}

		if pkgName != "" {
			return pkgName + "." + t.Name
		}

		// Check if this is a local type or an external type
		// If it's a type with uppercase first letter and no package context,
		// try to find the package it belongs to from imports
		if len(t.Name) > 0 && unicode.IsUpper(rune(t.Name[0])) {
			// Check if it's defined locally (has an Obj and is in current file)
			if t.Obj != nil && p.Package != "" {
				// Don't add package prefix to generic type parameters
				// Generic type parameters have Obj.Decl as *ast.Field (from type parameter list)
				// Regular types have Obj.Decl as *ast.TypeSpec (from type declarations)
				if _, isField := t.Obj.Decl.(*ast.Field); isField {
					return t.Name
				}
				return p.Package + "." + t.Name
			}

			// When pkgName is empty, use current package name for external types
			return t.Name
		}
		return t.Name
	case *ast.SelectorExpr:
		if pkgIdent, ok := t.X.(*ast.Ident); ok {
			return pkgIdent.Name + "." + t.Sel.Name
		}
	case *ast.IndexExpr:
		base := p.parseFieldType(t.X, pkgName)
		idx := p.parseFieldType(t.Index, pkgName)
		if base == "" || idx == "" {
			return ""
		}
		return base + "[" + idx + "]"
	case *ast.IndexListExpr:
		base := p.parseFieldType(t.X, pkgName)
		if base == "" {
			return ""
		}
		var parts []string
		for _, e := range t.Indices {
			s := p.parseFieldType(e, pkgName)
			if s == "" {
				return ""
			}
			parts = append(parts, s)
		}
		return base + "[" + strings.Join(parts, ", ") + "]"
	case *ast.StarExpr:
		// Recursively handle pointer types
		innerType := p.parseFieldType(t.X, pkgName)
		return "*" + innerType
	case *ast.ArrayType:
		// Handle slice types like []byte
		elementType := p.parseFieldType(t.Elt, pkgName)
		return "[]" + elementType
	case *ast.UnaryExpr:
		// Dereference address-of composite literals: &Type{}
		if t.Op == token.AND {
			if cl, ok := t.X.(*ast.CompositeLit); ok {
				return p.parseFieldType(cl.Type, pkgName)
			}
		}
		return p.parseFieldType(t.X, pkgName)
	case *ast.CompositeLit:
		// Return the type string of the composite literal
		return p.parseFieldType(t.Type, pkgName)
	}
	return "any"
}

// handleAnonymousEmbedding processes anonymous embedded fields and returns true if handled
func (p *File) handleAnonymousEmbedding(field *ast.Field, pkgName string, s *Struct) bool {
	switch t := field.Type.(type) {
	case *ast.Ident:
		// Local type embedding
		if t.Obj != nil {
			if ts, ok := t.Obj.Decl.(*ast.TypeSpec); ok {
				if st, ok := ts.Type.(*ast.StructType); ok {
					sub := p.processStructType(ts, st, pkgName)
					s.Fields = append(s.Fields, sub.Fields...)
					return true
				}
			}
		}
	case *ast.SelectorExpr:
		// External package type embedding
		if pkgIdent, ok := t.X.(*ast.Ident); ok {
			pkgAlias := pkgIdent.Name
			typeName := t.Sel.Name

			// Find the real package path
			pkgPath := pkgAlias
			for _, i := range p.Imports {
				if i.Name == pkgAlias {
					pkgPath = i.Path
				}
			}

			// Try to load the struct from the package
			if st, err := p.loadStructFromPackage(pkgPath, typeName); err == nil && st != nil {
				sub := p.processStructType(&ast.TypeSpec{Name: &ast.Ident{Name: typeName}}, st, pkgAlias)
				s.Fields = append(s.Fields, sub.Fields...)
				return true
			}
		}
	case *ast.StructType:
		// Anonymous inline struct embedding
		sub := p.processStructType(&ast.TypeSpec{Name: &ast.Ident{Name: "AnonymousStruct"}}, t, pkgName)
		s.Fields = append(s.Fields, sub.Fields...)
		return true
	}

	return false
}

// loadStructFromPackage loads a struct type definition from an external package by name
func (p *File) loadStructFromPackage(pkgPath, typeName string) (*ast.StructType, error) {
	modPath := findGoModDir(p.inputPath)
	cfg := &packages.Config{
		Mode: packages.NeedSyntax | packages.NeedTypes | packages.NeedImports,
		Dir:  modPath,
	}

	pkgs, err := packages.Load(cfg, pkgPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load package %q from %v: %w", pkgPath, modPath, err)
	}

	if len(pkgs) == 0 {
		return nil, fmt.Errorf("no packages found for path %q from %v", pkgPath, modPath)
	}

	for _, pkg := range pkgs {
		for _, syntax := range pkg.Syntax {
			for _, decl := range syntax.Decls {
				gen, ok := decl.(*ast.GenDecl)
				if !ok {
					continue
				}
				for _, spec := range gen.Specs {
					ts, ok := spec.(*ast.TypeSpec)
					if ok && ts.Name.Name == typeName {
						if st, ok := ts.Type.(*ast.StructType); ok {
							return st, nil
						}
					}
				}
			}
		}
	}

	return nil, fmt.Errorf("struct %s not found in package %s", typeName, pkgPath)
}
