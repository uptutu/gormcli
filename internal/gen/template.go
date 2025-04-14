package gen

var pkgTmpl = `package {{.Package}}

import (
	"gorm.io/gorm/g"
	{{range .Imports}}
		{{.Name}} {{.Path}}
	{{end}}
)

{{range .Interfaces}}
{{$IfaceName := .Name}}
func {{$IfaceName}}[T any](db *gorm.DB, opts ...g.Option) {{$IfaceName}}Interface[T] {
	return {{$IfaceName}}Impl[T]{
		Interface: g.G[T](db, opts...),
	}
}

type {{$IfaceName}}Interface[T any] interface {
	g.ChainInterface[T]
	{{range .Methods -}}
	{{.Name}}({{.ParamsString}}) ({{.ResultString}})
	{{end}}
}

type {{$IfaceName}}Impl[T any] struct {
	g.Interface[T]
}

{{range .Methods}}
func (e {{$IfaceName}}Impl[T]) {{.Name}}({{.ParamsString}}) ({{.ResultString}}) {
	{{.Body}}
}
{{end}}
{{end}}
`
