package snowflake_generator

import (
	"time"

	"github.com/bwmarrin/snowflake"
)

type Node struct {
	*snowflake.Node
}

var timeShift = snowflake.NodeBits + snowflake.StepBits

// GenerateForTimestamp Generate creates and returns a unique snowflake_generator ID
// To help guarantee uniqueness
// - Make sure your system is keeping accurate system time
// - Make sure you never have multiple nodes running with the same node ID
func (n *Node) GenerateForTimestamp(timestamp time.Time) snowflake.ID {

	id := n.Generate()

	val := int64(id) >> timeShift
	// preserve NodeBits & StepBits
	mask := int64((1 << timeShift) - 1)
	rightPart := int64(id) & mask

	// calculate time duration from the beginning of the date to old date we need create snowflake_generator id to
	idTime := time.UnixMilli(val + snowflake.Epoch)
	y, m, d := idTime.Date()
	diff := time.Date(y, m, d, 0, 0, 0, 0, time.UTC).UnixMilli() - timestamp.UnixMilli()

	// update ts & assemble back snowflake_generator.ID
	resultID := snowflake.ID(((val - diff) << timeShift) | rightPart)

	return resultID
}

func (n *Node) GenerateNewID(timestamp *time.Time) snowflake.ID {
	if timestamp == nil {
		return n.Generate()
	} else {
		return n.GenerateForTimestamp(*timestamp)
	}
}
