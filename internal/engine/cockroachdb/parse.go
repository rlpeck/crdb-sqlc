package cockroachdb

import (
	"io"
	"strings"

	"github.com/cockroachdb/cockroachdb-parser/pkg/sql/parser"
	"github.com/cockroachdb/cockroachdb-parser/pkg/sql/pgwire/pgerror"

	"github.com/rlpeck/crdb-sqlc/internal/source"
	"github.com/rlpeck/crdb-sqlc/internal/sql/ast"
	"github.com/rlpeck/crdb-sqlc/internal/sql/sqlerr"
)

func NewParser() *Parser {
	return &Parser{}
}

type Parser struct{}

func (p *Parser) Parse(r io.Reader) ([]ast.Statement, error) {
	blob, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	// CockroachDB cannot parse sqlc's @name parameter syntax (it overloads '@'),
	// so rewrite each @name to a length-preserving $N placeholder before parsing
	// and reconstruct the A_Expr("@") nodes during conversion. Because the
	// rewrite preserves byte offsets, the statement locations computed below are
	// valid against the original text that the compiler edits.
	contents, synth := rewriteAtParams(string(blob))

	parsed, err := parser.Parse(contents)
	if err != nil {
		return nil, normalizeErr(err)
	}

	var stmts []ast.Statement
	// cursor marks the start of the current statement's chunk: the byte right
	// after the previous statement's terminating semicolon. Mirroring the
	// PostgreSQL engine, a statement's range begins here so that a preceding
	// "-- name: ..." comment is part of the statement text.
	cursor := 0
	for i := range parsed {
		text := parsed[i].SQL
		if text == "" {
			continue
		}

		// Locate the (whitespace/comment/semicolon-trimmed) statement text
		// within the input to recover its absolute offset.
		rel := strings.Index(contents[cursor:], text)
		if rel < 0 {
			// Should not happen: the parser's per-statement SQL is a verbatim
			// slice of the input. Fall back to the cursor.
			rel = 0
		}
		innerStart := cursor + rel
		stmtEnd := innerStart + len(text)

		// Extend the chunk to just past the trailing semicolon (if any) so the
		// next statement's range starts cleanly after it.
		chunkEnd := len(contents)
		if semi := strings.IndexByte(contents[stmtEnd:], ';'); semi >= 0 {
			chunkEnd = stmtEnd + semi + 1
		}

		stmtLocation := cursor
		stmtLen := stmtEnd - cursor
		cursor = chunkEnd

		converter := &cc{loc: newLocator(text, innerStart), synth: synth}
		out := converter.convert(parsed[i].AST)
		if _, ok := out.(*ast.TODO); ok {
			continue
		}

		stmts = append(stmts, ast.Statement{
			Raw: &ast.RawStmt{
				Stmt:         out,
				StmtLocation: stmtLocation,
				StmtLen:      stmtLen,
			},
		})
	}
	return stmts, nil
}

// https://www.cockroachlabs.com/docs/stable/comments — CockroachDB follows
// PostgreSQL comment syntax.
func (p *Parser) CommentSyntax() source.CommentSyntax {
	return source.CommentSyntax{
		Dash:      true,
		SlashStar: true,
	}
}

func normalizeErr(err error) error {
	if err == nil {
		return nil
	}
	// cockroachdb-parser surfaces syntax errors as pgerror codes. Map the
	// message onto sqlc's error type. The parser does not expose a stable SQL
	// byte offset for the error, so we leave Location/Line unset (the message
	// itself includes an "at or near" hint).
	if pgErr := pgerror.Flatten(err); pgErr != nil {
		return &sqlerr.Error{
			Message: pgErr.Message,
			Code:    pgErr.Code,
		}
	}
	return err
}
