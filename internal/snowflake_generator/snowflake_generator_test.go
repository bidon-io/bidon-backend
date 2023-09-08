package snowflake_generator

import (
	"github.com/bwmarrin/snowflake"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestSnowFlake(t *testing.T) {
	nodeSnf, err := snowflake.NewNode(1)
	assert.NoError(t, err)
	node := Node{nodeSnf}
	date, _ := time.Parse(time.DateOnly, "2020-01-01")
	id := node.GenerateForTimestamp(date)

	assert.NotEqual(t, 0, id)
	assert.EqualValues(t, "2020-01-01", time.UnixMilli(id.Time()).Format(time.DateOnly), "ID should have set date 2020-01-01")
}
