package db

import (
	"gorm.io/gorm"
)

// AfterCreate hook to set PublicUID and ensure uniqueness
func (l *LineItem) AfterCreate(tx *gorm.DB) (err error) {
	return GenerateUniquePublicUID(tx, l)
}
