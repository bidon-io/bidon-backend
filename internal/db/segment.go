package db

import (
	"gorm.io/gorm"
)

// AfterCreate hook to set PublicUID and ensure uniqueness
func (s *Segment) AfterCreate(tx *gorm.DB) (err error) {
	return GenerateUniquePublicUID(tx, s)
}
