package cockroachdb

import (
	"fmt"
	"strings"

	"github.com/cockroachdb/cockroachdb-parser/pkg/sql/lexbase"
)

// IsReservedKeyword reports whether s is a fully-reserved CockroachDB keyword.
// CockroachDB's lexer classifies keywords into categories; category "R" marks
// the reserved set (keywords that cannot be used as a bare identifier).
func (p *Parser) IsReservedKeyword(s string) bool {
	return lexbase.KeywordsCategories[strings.ToLower(s)] == "R"
}

// The methods below implement format.Dialect. CockroachDB follows PostgreSQL's
// quoting, placeholder, and cast syntax.

func hasMixedCase(s string) bool {
	for _, r := range s {
		if r >= 'A' && r <= 'Z' {
			return true
		}
	}
	return false
}

// QuoteIdent returns a quoted identifier if it needs quoting.
func (p *Parser) QuoteIdent(s string) string {
	if p.IsReservedKeyword(s) || hasMixedCase(s) {
		return `"` + s + `"`
	}
	return s
}

// TypeName returns the SQL type name for the given namespace and name.
func (p *Parser) TypeName(ns, name string) string {
	if ns != "" {
		return ns + "." + name
	}
	return name
}

// Param returns the parameter placeholder for the given number ($1, $2, ...).
func (p *Parser) Param(n int) string {
	return fmt.Sprintf("$%d", n)
}

// NamedParam returns the named parameter placeholder for the given name.
func (p *Parser) NamedParam(name string) string {
	return "@" + name
}

// Cast returns a type cast expression using PostgreSQL's expr::type syntax.
func (p *Parser) Cast(arg, typeName string) string {
	return arg + "::" + typeName
}
