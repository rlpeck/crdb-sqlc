package clickhouse

import (
	"github.com/rlpeck/crdb-sqlc/internal/sql/catalog"
)

func NewCatalog() *catalog.Catalog {
	def := "default" // ClickHouse default database
	return &catalog.Catalog{
		DefaultSchema: def,
		Schemas: []*catalog.Schema{
			defaultSchema(def),
		},
		Extensions: map[string]struct{}{},
	}
}
