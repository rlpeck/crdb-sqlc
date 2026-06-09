package cockroachdb

import (
	"github.com/sqlc-dev/sqlc/internal/sql/catalog"
)

// NewCatalog returns a CockroachDB-specific catalog. CockroachDB uses the
// "public" schema by default and is largely PostgreSQL-compatible, but the type
// vocabulary (STRING, INT8, BYTES, ...) and built-in functions are seeded here
// independently so the engine can diverge from PostgreSQL as needed.
func NewCatalog() *catalog.Catalog {
	def := "public"
	return &catalog.Catalog{
		DefaultSchema: def,
		Schemas: []*catalog.Schema{
			defaultSchema(def),
		},
		Extensions: map[string]struct{}{},
	}
}
