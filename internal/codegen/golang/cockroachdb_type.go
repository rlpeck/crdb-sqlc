package golang

import (
	"log"

	"github.com/sqlc-dev/sqlc/internal/codegen/golang/opts"
	"github.com/sqlc-dev/sqlc/internal/codegen/sdk"
	"github.com/sqlc-dev/sqlc/internal/debug"
	"github.com/sqlc-dev/sqlc/internal/plugin"
)

// cockroachType maps CockroachDB column types to Go types. CockroachDB is
// PostgreSQL-compatible but uses its own canonical type spellings (STRING, INT8,
// BYTES, ...), which is why it has a dedicated mapping rather than reusing the
// PostgreSQL one. The mapping favors database/sql standard types so it works
// with both lib/pq and pgx; it covers the common types and falls back to
// interface{} for anything unrecognized.
func cockroachType(req *plugin.GenerateRequest, options *opts.Options, col *plugin.Column) string {
	columnType := sdk.DataType(col.Type)
	notNull := col.NotNull || col.IsArray

	switch columnType {

	case "string", "text", "varchar", "char", "bpchar", "name", "collatedstring":
		if notNull {
			return "string"
		}
		return "sql.NullString"

	case "bool", "boolean":
		if notNull {
			return "bool"
		}
		return "sql.NullBool"

	case "int8", "int", "integer", "bigint", "serial", "serial8", "bigserial":
		if notNull {
			return "int64"
		}
		return "sql.NullInt64"

	case "int4", "serial4":
		if notNull {
			return "int32"
		}
		return "sql.NullInt32"

	case "int2", "smallint", "serial2", "smallserial":
		if notNull {
			return "int16"
		}
		return "sql.NullInt16"

	case "float8", "float", "double precision", "real", "float4":
		if notNull {
			return "float64"
		}
		return "sql.NullFloat64"

	case "decimal", "numeric", "dec":
		// CockroachDB DECIMAL has arbitrary precision; surface it as a string to
		// avoid lossy float conversions.
		if notNull {
			return "string"
		}
		return "sql.NullString"

	case "bytes", "bytea", "blob":
		return "[]byte"

	case "uuid":
		if notNull {
			return "string"
		}
		return "sql.NullString"

	case "date", "time", "timetz", "timestamp", "timestamptz":
		if notNull {
			return "time.Time"
		}
		return "sql.NullTime"

	case "interval":
		if notNull {
			return "string"
		}
		return "sql.NullString"

	case "json", "jsonb":
		return "[]byte"

	case "inet":
		if notNull {
			return "string"
		}
		return "sql.NullString"

	case "oid", "regclass", "regproc", "regtype":
		if notNull {
			return "int64"
		}
		return "sql.NullInt64"

	case "any", "anyelement":
		return "interface{}"

	default:
		if debug.Active {
			log.Printf("unknown CockroachDB type: %s\n", columnType)
		}
		return "interface{}"
	}
}
