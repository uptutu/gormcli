package gen

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"golang.org/x/tools/imports"
)

type Generator struct {
	Name  string
	Files []*File
}

func (g *Generator) Process(input, output string) error {
	info, err := os.Stat(input)
	if err != nil {
		return err
	}

	if info.IsDir() {
		return filepath.Walk(input, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				return g.processFile(path, output)
			}
			return nil
		})
	}
	return g.processFile(input, output)
}

func (g *Generator) Gen() error {
	tmpl, err := template.New("").Parse(pkgTmpl)
	if err != nil {
		panic(err)
	}

	filesName := map[string]bool{}
	for _, file := range g.Files {
		os.MkdirAll(file.outputPath, 0o755)
		outputName := filepath.Base(file.inputPath)

		counter := 1
		for {
			if _, exists := filesName[outputName]; !exists {
				break
			}
			ext := filepath.Ext(outputName)
			base := strings.TrimSuffix(outputName, ext)
			outputName = fmt.Sprintf("%s_%d%s", base, counter, ext)
			counter++
		}

		filesName[outputName] = true
		outputFile := filepath.Join(file.outputPath, outputName)
		f, err := os.Create(outputFile)
		if err != nil {
			panic(fmt.Sprintf("failed to create file %v, got error %v", outputFile, err))
		}

		fmt.Printf("Generating file %s from %s...\n", outputFile, file.inputPath)
		err = tmpl.Execute(f, file)
		if err != nil {
			panic(fmt.Sprintf("failed to render template %v, got error %v", file.inputPath, err))
		}

		result, err := imports.Process(outputFile, nil, nil)
		os.WriteFile(outputFile, result, 0o640)
	}
	return nil
}

func (g *Generator) processFile(inputFile, outFile string) error {
	inputFile, err := filepath.Abs(inputFile)
	if err != nil {
		return err
	}

	fileset := token.NewFileSet()
	f, err := parser.ParseFile(fileset, inputFile, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("can't parse file %q: %s", inputFile, err)
	}

	file := &File{Package: g.Name, inputPath: inputFile, outputPath: outFile}
	g.Files = append(g.Files, file)

	ast.Walk(file, f)

	return nil
}

type (
	File struct {
		Package    string
		inputPath  string
		outputPath string
		Imports    []Import
		Interfaces []Interface
	}
	Import struct {
		Name string
		Path string
	}
	Interface struct {
		Name    string
		Doc     string
		Methods []*Method
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
)

func (p Param) GoFullType() string {
	return p.Type
}

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

func (m Method) ResultString() string {
	if m.SQL.Raw != "" {
		var rets []string
		for _, r := range m.Result {
			rets = append(rets, r.GoFullType())
		}

		return strings.Join(rets, ", ")
	}
	return fmt.Sprintf("%sInterface[T]", m.Interface.Name)
}

func (m Method) Body() string {
	if m.SQL.Raw != "" {
		return m.finishMethodBody()
	}
	return m.chainMethodBody()
}

func (m Method) processSQL(sql string) string {
	sqlSnippet, err := RenderSQLTemplate(sql)
	if err != nil {
		panic(fmt.Sprintf("Failed to parsing SQL template for %s.%s %q: %v", m.Interface.Name, m.Name, m.SQL, err))
	}

	return sqlSnippet
}

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

func (m Method) parseParams(fields *ast.FieldList) []Param {
	if fields == nil {
		return nil
	}

	var parseExprType func(e ast.Expr) string
	parseExprType = func(expr ast.Expr) string {
		switch t := expr.(type) {
		case *ast.Ident:
			return t.Name
		case *ast.SelectorExpr:
			// e.g. models.User
			return parseExprType(t.X) + "." + t.Sel.Name
		case *ast.ArrayType:
			// slice type: "[]" + element type
			return "[]" + parseExprType(t.Elt)
		case *ast.StarExpr:
			// pointer type: "*" + underlying type
			return "*" + parseExprType(t.X)
		default:
			return fmt.Sprintf("unknown")
		}
	}

	var params []Param
	for _, field := range fields.List {
		typ := parseExprType(field.Type)

		names := field.Names
		if len(names) == 0 {
			names = []*ast.Ident{{Name: ""}}
		}

		for _, n := range names {
			params = append(params, Param{
				Name: n.Name,
				Type: typ,
			})
		}
	}

	return params
}

func (p *File) Visit(n ast.Node) (w ast.Visitor) {
	switch n := n.(type) {
	case *ast.ImportSpec:
		importName, _ := strconv.Unquote(n.Path.Value)
		importName = path.Base(importName)
		if n.Name != nil {
			importName = n.Name.Name
		}

		p.Imports = append(p.Imports, Import{
			Name: importName,
			Path: n.Path.Value,
		})
	case *ast.TypeSpec:
		if data, ok := n.Type.(*ast.InterfaceType); ok {
			r := Interface{
				Name: n.Name.Name,
				Doc:  n.Doc.Text(),
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

					method.Params = method.parseParams(m.Type.(*ast.FuncType).Params)
					method.Result = method.parseParams(m.Type.(*ast.FuncType).Results)

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

			p.Interfaces = append(p.Interfaces, r)
		}
	}
	return p
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
