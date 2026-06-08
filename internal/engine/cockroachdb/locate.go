package cockroachdb

import (
	"strings"

	"github.com/cockroachdb/cockroachdb-parser/pkg/sql/lexbase"
	"github.com/cockroachdb/cockroachdb-parser/pkg/sql/scanner"
)

// The cockroachdb-parser tree AST does not carry source byte offsets, but sqlc
// needs accurate offsets to rewrite sqlc.arg() calls into $N placeholders and
// to expand SELECT *. We recover those offsets by re-scanning each statement
// with CockroachDB's lexer (which does report token positions) and matching the
// tokens that matter.
//
// All offsets produced by the locator are absolute (relative to the full input)
// because base is added to every token position.

// scanSym is a minimal implementation of scanner.ScanSymType used to drive the
// CockroachDB SQL scanner and capture each token's id, start position, and text.
type scanSym struct {
	id  int32
	pos int32
	str string
	val interface{}
}

func (s *scanSym) ID() int32                 { return s.id }
func (s *scanSym) SetID(id int32)            { s.id = id }
func (s *scanSym) Pos() int32                { return s.pos }
func (s *scanSym) SetPos(p int32)            { s.pos = p }
func (s *scanSym) Str() string               { return s.str }
func (s *scanSym) SetStr(v string)           { s.str = v }
func (s *scanSym) UnionVal() interface{}     { return s.val }
func (s *scanSym) SetUnionVal(v interface{}) { s.val = v }

type token struct {
	id  int32
	pos int // offset within the statement SQL
	str string
}

const (
	tokDot    = int32('.')
	tokLParen = int32('(')
	tokStar   = int32('*')
	tokComma  = int32(',')
)

// paramLoc records the absolute offsets of a sqlc.arg()/narg()/slice() call:
// funcAbs points at the leading "sqlc" identifier, argAbs at the argument name.
type paramLoc struct {
	funcAbs int
	argAbs  int
}

type locator struct {
	base   int
	toks   []token
	params map[string][]paramLoc
	stars  []int  // absolute offsets of select-list star targets, in source order
	used   []bool // parallel to stars
}

func newLocator(sql string, base int) *locator {
	l := &locator{
		base:   base,
		toks:   scanTokens(sql),
		params: map[string][]paramLoc{},
	}
	l.index()
	return l
}

func scanTokens(sql string) []token {
	var sc scanner.SQLScanner
	sc.Init(sql)
	var toks []token
	for {
		var sym scanSym
		sc.Scan(&sym)
		if sym.id == 0 || sym.id == lexbase.ERROR {
			break
		}
		toks = append(toks, token{id: sym.id, pos: int(sym.pos), str: sym.str})
	}
	return toks
}

func paramKey(kind, name string) string { return kind + "\x00" + name }

// index walks the token stream once, recording the locations of sqlc.* calls
// and select-list stars.
func (l *locator) index() {
	t := l.toks
	for i := 0; i < len(t); i++ {
		// sqlc.arg(name) / sqlc.narg(name) / sqlc.slice(name)
		if t[i].id == lexbase.IDENT && strings.EqualFold(t[i].str, "sqlc") &&
			i+4 < len(t) &&
			t[i+1].id == tokDot &&
			t[i+2].id == lexbase.IDENT && isSqlcParamFunc(t[i+2].str) &&
			t[i+3].id == tokLParen &&
			t[i+4].id == lexbase.IDENT {
			kind := strings.ToLower(t[i+2].str)
			name := t[i+4].str
			key := paramKey(kind, name)
			l.params[key] = append(l.params[key], paramLoc{
				funcAbs: l.base + t[i].pos,
				argAbs:  l.base + t[i+4].pos,
			})
			continue
		}

		// Select-list and RETURNING stars: a `*` whose preceding significant
		// token is SELECT, DISTINCT, RETURNING, or a comma. This excludes
		// count(*) (preceded by `(`) and multiplication (preceded by an operand).
		if t[i].id == tokStar && i > 0 {
			prev := t[i-1]
			if prev.id == tokComma ||
				strings.EqualFold(prev.str, "select") ||
				strings.EqualFold(prev.str, "distinct") ||
				strings.EqualFold(prev.str, "returning") {
				l.stars = append(l.stars, l.base+t[i].pos)
				l.used = append(l.used, false)
			}
		}
	}
}

func isSqlcParamFunc(s string) bool {
	switch strings.ToLower(s) {
	case "arg", "narg", "slice":
		return true
	}
	return false
}

// popParam returns the offsets for the next unconsumed sqlc.<kind>(<name>) call.
// Because every such call with the same (kind, name) is replaced by identical
// text, the order in which equal calls are consumed does not affect the output.
func (l *locator) popParam(kind, name string) (paramLoc, bool) {
	key := paramKey(strings.ToLower(kind), name)
	locs := l.params[key]
	if len(locs) == 0 {
		return paramLoc{}, false
	}
	l.params[key] = locs[1:]
	return locs[0], true
}

// popBareStar returns the offset of the next unconsumed bare select-list star.
func (l *locator) popBareStar() (int, bool) {
	for i, used := range l.used {
		if !used {
			l.used[i] = true
			return l.stars[i], true
		}
	}
	return 0, false
}

// qualifiedStar returns the absolute offset of `<table>.*`, pointing at the
// leading table identifier (the position SELECT * expansion expects).
func (l *locator) qualifiedStar(table string) (int, bool) {
	t := l.toks
	for i := 0; i+2 < len(t); i++ {
		if t[i].id == lexbase.IDENT && strings.EqualFold(t[i].str, table) &&
			t[i+1].id == tokDot && t[i+2].id == tokStar {
			return l.base + t[i].pos, true
		}
	}
	return 0, false
}
