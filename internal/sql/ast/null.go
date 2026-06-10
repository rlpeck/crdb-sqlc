package ast

import "github.com/rlpeck/crdb-sqlc/internal/sql/format"

type Null struct {
}

func (n *Null) Pos() int {
	return 0
}
func (n *Null) Format(buf *TrackedBuffer, d format.Dialect) {
	buf.WriteString("NULL")
}
