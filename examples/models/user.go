package models

import "gorm.io/gorm"

type User struct {
	gorm.Model
	Name    string
	Age     int
	Role    string
	IsAdult bool `gorm:"column:is_adult"`
}
