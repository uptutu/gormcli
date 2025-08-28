package gen

import (
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"
)

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
	modPath := findGoModDir(filename)
	cfg := &packages.Config{
		Mode: packages.NeedName,
		Dir:  modPath,
	}

	// Load the package containing the current file
	pkgs, err := packages.Load(cfg, filepath.Dir(filename))
	if err != nil {
		return ""
	}

	if len(pkgs) == 0 {
		return ""
	}

	return pkgs[0].PkgPath
}
