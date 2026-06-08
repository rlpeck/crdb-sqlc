package cockroachdb

import (
	"github.com/sqlc-dev/sqlc/internal/sql/ast"
	"github.com/sqlc-dev/sqlc/internal/sql/catalog"
)

func typeName(name string) *ast.TypeName { return &ast.TypeName{Name: name} }

func arg(name, typ string) *catalog.Argument {
	return &catalog.Argument{Name: name, Type: typeName(typ)}
}

// defaultSchema seeds a CockroachDB schema with a focused set of commonly used
// built-in functions. This is intentionally not exhaustive; it covers the
// aggregates and scalar functions that most frequently appear in queries and
// can be grown over time. Type resolution for table columns flows from CREATE
// TABLE definitions, so this list only needs to describe function signatures.
func defaultSchema(name string) *catalog.Schema {
	s := &catalog.Schema{Name: name}
	s.Funcs = []*catalog.Function{
		// Aggregates
		{Name: "count", Args: []*catalog.Argument{}, ReturnType: typeName("int8")}, // count(*)
		{Name: "count", Args: []*catalog.Argument{arg("", "any")}, ReturnType: typeName("int8")},
		{Name: "sum", Args: []*catalog.Argument{arg("", "int8")}, ReturnType: typeName("decimal")},
		{Name: "avg", Args: []*catalog.Argument{arg("", "int8")}, ReturnType: typeName("decimal")},
		{Name: "min", Args: []*catalog.Argument{arg("", "any")}, ReturnType: typeName("any")},
		{Name: "max", Args: []*catalog.Argument{arg("", "any")}, ReturnType: typeName("any")},
		{Name: "bool_and", Args: []*catalog.Argument{arg("", "bool")}, ReturnType: typeName("bool")},
		{Name: "bool_or", Args: []*catalog.Argument{arg("", "bool")}, ReturnType: typeName("bool")},
		{Name: "string_agg", Args: []*catalog.Argument{arg("", "string"), arg("", "string")}, ReturnType: typeName("string")},
		{Name: "array_agg", Args: []*catalog.Argument{arg("", "any")}, ReturnType: typeName("any")},

		// String functions
		{Name: "lower", Args: []*catalog.Argument{arg("", "string")}, ReturnType: typeName("string")},
		{Name: "upper", Args: []*catalog.Argument{arg("", "string")}, ReturnType: typeName("string")},
		{Name: "length", Args: []*catalog.Argument{arg("", "string")}, ReturnType: typeName("int8")},
		{Name: "char_length", Args: []*catalog.Argument{arg("", "string")}, ReturnType: typeName("int8")},
		{Name: "trim", Args: []*catalog.Argument{arg("", "string")}, ReturnType: typeName("string")},
		{Name: "ltrim", Args: []*catalog.Argument{arg("", "string")}, ReturnType: typeName("string")},
		{Name: "rtrim", Args: []*catalog.Argument{arg("", "string")}, ReturnType: typeName("string")},
		{Name: "substr", Args: []*catalog.Argument{arg("", "string"), arg("", "int8")}, ReturnType: typeName("string")},
		{Name: "concat", Args: []*catalog.Argument{arg("", "any")}, ReturnType: typeName("string")},

		// Time functions
		{Name: "now", Args: []*catalog.Argument{}, ReturnType: typeName("timestamptz")},
		{Name: "current_timestamp", Args: []*catalog.Argument{}, ReturnType: typeName("timestamptz")},
		{Name: "current_date", Args: []*catalog.Argument{}, ReturnType: typeName("date")},

		// Identifier / misc functions
		{Name: "gen_random_uuid", Args: []*catalog.Argument{}, ReturnType: typeName("uuid")},
		{Name: "uuid_v4", Args: []*catalog.Argument{}, ReturnType: typeName("bytes")},
		{Name: "abs", Args: []*catalog.Argument{arg("", "int8")}, ReturnType: typeName("int8")},
		{Name: "coalesce", Args: []*catalog.Argument{arg("", "any"), arg("", "any")}, ReturnType: typeName("any")},
	}
	return s
}
