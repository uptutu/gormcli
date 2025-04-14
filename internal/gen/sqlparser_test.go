package gen

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

var methodExpectedLines = map[string][]string{
	"GetByID": {
		"var sb strings.Builder",
		`sb.WriteString("SELECT * FROM @@table WHERE id=@id ")`,
	},
	"FilterWithColumn": {
		"var sb strings.Builder",
		`sb.WriteString("SELECT * FROM @@table WHERE @@column=@value ")`,
	},
	"QueryWith": {
		"var sb strings.Builder",
		`sb.WriteString("SELECT * FROM users ")`,
		`if user.ID > 0 {`,
		`sb.WriteString("WHERE id=@user.ID ")`,
		`} else if user.Name != "" {`,
		`sb.WriteString("WHERE username=@user.Name ")`,
		`}`,
	},
	"Update": {
		"var sb strings.Builder",
		`sb.WriteString("UPDATE @@table ")`,
		`{`,
		`var tmp strings.Builder`,
		`if user.Name != "" {`,
		`tmp.WriteString("username=@user.Name, ")`,
		`}`,
		`if user.Age > 0 {`,
		`tmp.WriteString("age=@user.Age, ")`,
		`}`,
		`if user.Age >= 18 {`,
		`tmp.WriteString("is_adult=1 ")`,
		`} else {`,
		`tmp.WriteString("is_adult=0 ")`,
		`}`,
		`c := strings.TrimSpace(tmp.String())`,
		`if c != "" {`,
		`if strings.HasSuffix(c, ",") {`,
		`c = strings.TrimRight(c, ",")`,
		`c = strings.TrimSpace(c)`,
		`}`,
		`sb.WriteString("SET ")`,
		`sb.WriteString(c)`,
		`}`,
		`}`,
		`sb.WriteString("WHERE id=@id ")`,
	},
	"Filter": {
		"var sb strings.Builder",
		`sb.WriteString("SELECT * FROM @@table ")`,
		`{`,
		`var tmp strings.Builder`,
		`for _,user:=range users {`,
		`if user.Name !="" && user.Age >0 {`,
		`tmp.WriteString("(username = @user.Name AND age=@user.Age AND role LIKE concat(\"%\",@user.Role,\"%\")) OR ")`,
		`}`,
		`}`,
		`c := strings.TrimSpace(tmp.String())`,
		`if c != "" {`,
		`sb.WriteString("WHERE ")`,
		`sb.WriteString(c)`,
		`}`,
		`}`,
	},
	"FilterByNameAndAge": {
		"var sb strings.Builder",
		`sb.WriteString("where(\"name=@name AND age=@age\") ")`,
	},
	"FilterWithTime": {
		"var sb strings.Builder",
		`sb.WriteString("SELECT * FROM @@table ")`,
		`{`,
		`var tmp strings.Builder`,
		`if !start.IsZero() {`,
		`tmp.WriteString("created_time > @start ")`,
		`}`,
		`if !end.IsZero() {`,
		`tmp.WriteString("AND created_time < @end ")`,
		`}`,
		`c := strings.TrimSpace(tmp.String())`,
		`if c != "" {`,
		`sb.WriteString("WHERE ")`,
		`sb.WriteString(c)`,
		`}`,
		`}`,
	},
}

// TestRenderSQLTemplate
func TestRenderSQLTemplate(t *testing.T) {
	const queryFilePath = "../../examples/query.go"

	fset := token.NewFileSet()
	parsedFile, err := parser.ParseFile(fset, queryFilePath, nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("failed to parse %s: %v", queryFilePath, err)
	}

	var queryInterface *ast.InterfaceType
	for _, decl := range parsedFile.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range genDecl.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			if ts.Name.Name == "Query" {
				if iface, ok := ts.Type.(*ast.InterfaceType); ok {
					queryInterface = iface
					break
				}
			}
		}
	}

	if queryInterface == nil {
		t.Fatalf("did not find Query[T any] interface in %s", queryFilePath)
	}

	for _, method := range queryInterface.Methods.List {
		name := method.Names[0].Name

		doc := extractSQL(method.Doc.Text(), name)
		if doc.Raw == "" && doc.Where == "" && doc.Select == "" {
			t.Fatalf("[SKIP] method %s has no doc", name)
			continue
		}

		if doc.Raw == "" {
			continue
		}

		got, err := RenderSQLTemplate(doc.Raw)
		t.Run(name, func(t *testing.T) {
			if err != nil {
				t.Fatalf("RenderSQLTemplate error for method %s: %v\nDoc:\n%s", name, err, doc)
			}

			gotLines := splitNonEmptyLines(got)
			wantLines := methodExpectedLines[name]

			if len(gotLines) != len(wantLines) {
				t.Errorf("line count mismatch for %s:\n gotLines=%d\n wantLines=%d\n---got---\n%v\n---want---\n%v\n",
					name, len(gotLines), len(wantLines),
					strings.Join(gotLines, "\n"), strings.Join(wantLines, "\n"))
				return
			}

			for i := range wantLines {
				gotLine := strings.TrimSpace(gotLines[i])
				wantLine := strings.TrimSpace(wantLines[i])
				if gotLine != wantLine {
					t.Errorf("method %s line %d mismatch:\n got:  %q\n want: %q", name, i+1, gotLine, wantLine)
				}
			}
		})
	}
}

func splitNonEmptyLines(s string) []string {
	var out []string
	for _, l := range strings.Split(s, "\n") {
		l = strings.TrimSpace(l)
		if l != "" {
			out = append(out, l)
		}
	}
	return out
}
