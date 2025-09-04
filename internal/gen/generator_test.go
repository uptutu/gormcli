package gen

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestGeneratorWithQueryInterface(t *testing.T) {
	inputPath, err := filepath.Abs("../../examples/query.go")
	if err != nil {
		t.Fatalf("failed to get absolute path: %v", err)
	}

	goldenPath, err := filepath.Abs("../../examples/output/query.go")
	if err != nil {
		t.Fatalf("failed to get absolute output path: %v", err)
	}

	outputDir := t.TempDir()

	g := &Generator{Name: "examples"}

	if err := g.Process(inputPath, outputDir); err != nil {
		t.Fatalf("Process error: %v", err)
	}
	if err := g.Gen(); err != nil {
		t.Fatalf("Gen error: %v", err)
	}

	files, err := os.ReadDir(outputDir)
	if err != nil {
		t.Fatalf("failed to read output dir: %v", err)
	}
	if len(files) == 0 {
		t.Fatalf("no files were generated in %s", outputDir)
	}

	goldenBytes, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("failed to read golden file %s: %v", goldenPath, err)
	}
	goldenStr := string(goldenBytes)

	generatedFile := filepath.Join(outputDir, files[0].Name())
	genBytes, err := os.ReadFile(generatedFile)
	if err != nil {
		t.Fatalf("failed to read generated file %s: %v", generatedFile, err)
	}
	generatedStr := string(genBytes)

	if _, err := parser.ParseFile(token.NewFileSet(), generatedFile, genBytes, parser.AllErrors); err != nil {
		t.Errorf("generated code %s has invalid Go syntax: %v", generatedFile, err)
	}

	if goldenStr != generatedStr {
		t.Errorf("generated file differs from golden file\nGOLDEN: %s\nGENERATED: %s\n%s",
			goldenPath, generatedFile, generatedStr)
	}
}

func TestProcessStructType(t *testing.T) {
	fileset := token.NewFileSet()
	file, err := parser.ParseFile(fileset, "../../examples/models/user.go", nil, parser.AllErrors)
	if err != nil {
		t.Fatalf("failed to parse file: %v", err)
	}

	var structType *ast.StructType

	ast.Inspect(file, func(n ast.Node) bool {
		typeSpec, ok := n.(*ast.TypeSpec)
		if ok && typeSpec.Name.Name == "User" {
			structType = typeSpec.Type.(*ast.StructType)
			return false
		}
		return true
	})

	if structType == nil {
		t.Fatalf("failed to find User struct")
	}

	expected := Struct{
		Name: "User",
		Fields: []Field{
			{Name: "ID", DBName: "id", GoType: "uint"},
			{Name: "CreatedAt", DBName: "created_at", GoType: "time.Time"},
			{Name: "UpdatedAt", DBName: "updated_at", GoType: "time.Time"},
			{Name: "DeletedAt", DBName: "deleted_at", GoType: "gorm.DeletedAt"},
			{Name: "Name", DBName: "name", GoType: "string"},
			{Name: "Age", DBName: "age", GoType: "int"},
			{Name: "Role", DBName: "role", GoType: "string"},
		},
	}

	p := File{
		Imports: []Import{
			{Name: "gorm", Path: "gorm.io/gorm"},
		},
	}

	result := p.processStructType(&ast.TypeSpec{Name: &ast.Ident{Name: "User"}}, structType, "")
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected %+v, got %+v", expected, result)
	}
}
