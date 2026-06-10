package cockroachdb

import (
	"log"
	"strings"

	"github.com/rlpeck/crdb-sqlc/internal/debug"
	"github.com/rlpeck/crdb-sqlc/internal/sql/ast"
)

func todo(prefix string, n interface{}) *ast.TODO {
	if debug.Active {
		log.Printf("cockroachdb.convert: %s: unhandled node type %T\n", prefix, n)
	}
	return &ast.TODO{}
}

// identifier lowercases an identifier. CockroachDB, like PostgreSQL, folds
// unquoted identifiers to lower case.
func identifier(id string) string {
	return strings.ToLower(id)
}

// NewIdentifier builds an ast.String suitable for use inside a ColumnRef or
// Funcname list.
func NewIdentifier(t string) *ast.String {
	return &ast.String{Str: identifier(t)}
}
