package db

import (
	"fmt"
	"github.com/bwmarrin/snowflake"
	"gorm.io/gorm"
)

// AfterCreate hook to set PublicUID and ensure uniqueness
func (s *Segment) AfterCreate(tx *gorm.DB) (err error) {
	// Attempt to generate a unique PublicUID
	for {
		snowflakeNode, ok := tx.InstanceGet("snowflakeNode")
		if !ok {
			return fmt.Errorf("error reading snowflakeNode from gorm instance")
		}
		generatedUID := snowflakeNode.(*snowflake.Node).Generate().Int64()

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
			break
		}
		// If not unique, regenerate the PublicUID and try again
	}

	return nil
}
