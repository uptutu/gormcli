package twolevel

import (
	s "gorm.io/cmd/gorm/examples/filters/twolevel/nested"
	"gorm.io/cmd/gorm/genconfig"
)

// Parent config excludes nested.I2 and nested.S2 across this directory’s subtree
var _ = genconfig.Config{
	ExcludeInterfaces: []any{s.I2[any](nil)},
	ExcludeStructs:    []any{s.S2{}},
}
