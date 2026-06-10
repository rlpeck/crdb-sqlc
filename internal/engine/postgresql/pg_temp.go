package postgresql

import (
	"github.com/rlpeck/crdb-sqlc/internal/sql/catalog"
)

func pgTemp() *catalog.Schema {
	return &catalog.Schema{Name: "pg_temp"}
}
