package db

import (
	"database/sql"
	"gorm.io/gorm"
)

// BeforeCreate hook to set PublicUID
func (l *LineItem) BeforeCreate(tx *gorm.DB) (err error) {
	publicUID, err := generatePublicUID(tx)
	if err != nil {
		return err
	}

	l.PublicUID = &sql.NullInt64{Int64: publicUID, Valid: true}
	return nil
}
