package pattern

import "gorm.io/cmd/gorm/genconfig"

// Include only interfaces whose names start with "Query"
var _ = genconfig.Config{
	IncludeInterfaces: []any{"Query*"},
}
