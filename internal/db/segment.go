package db

import (
	"fmt"
	"github.com/bwmarrin/snowflake"
	"gorm.io/gorm"
)

// AfterCreate hook to set PublicUID and ensure uniqueness
func (s *Segment) AfterCreate(tx *gorm.DB) (err error) {
	passedSnowflakeNode, ok := tx.Get("snowflakeNode")
	if !ok {
		return fmt.Errorf("error reading snowflakeNode from gorm")
	}
	snowflakeNode, ok := passedSnowflakeNode.(*snowflake.Node)
	if !ok {
		return fmt.Errorf("error converting snowflakeNode from gorm instance")
	}

	// Attempt to generate a unique PublicUID with a maximum of 10 attempts
	maxAttempts := 10
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		generatedUID := snowflakeNode.Generate().Int64()

		// Check uniqueness
		var count int64
		if err := tx.Model(&Segment{}).Where("public_uid = ?", generatedUID).Count(&count).Error; err != nil {
			return err
		}

		if count == 0 {
			// If unique, update the record with the generated PublicUID and break the loop
			if err := tx.Model(s).Update("public_uid", generatedUID).Error; err != nil {
				return err
			}
			return nil
		}
		// If not unique, regenerate the PublicUID and try again, up to the maximum attempts
	}

	return fmt.Errorf("failed to generate a unique PublicUID after %d attempts", maxAttempts)
}
