package blacklist

import (
	"gorm.io/cmd/gorm/genconfig"
)

// Exclude I2 and S2
var _ = genconfig.Config{
	ExcludeInterfaces: []any{I2[any](nil)},
	ExcludeStructs:    []any{S2{}},
}
