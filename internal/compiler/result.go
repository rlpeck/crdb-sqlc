package compiler

import (
	"github.com/rlpeck/crdb-sqlc/internal/sql/catalog"
)

type Result struct {
	Catalog *catalog.Catalog
	Queries []*Query
}
