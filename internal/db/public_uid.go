package db

import (
	"fmt"
	"github.com/bwmarrin/snowflake"
	"gorm.io/gorm"
)

func generatePublicUID(tx *gorm.DB) (int64, error) {
	passedSnowflakeNode, ok := tx.Get("snowflakeNode")
	if !ok {
		return 0, fmt.Errorf("error reading snowflakeNode from gorm")
	}
	snowflakeNode, ok := passedSnowflakeNode.(*snowflake.Node)
	if !ok {
		return 0, fmt.Errorf("error converting snowflakeNode from gorm")
	}

	return snowflakeNode.Generate().Int64(), nil
}
