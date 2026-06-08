package cockroachdb

import (
	"strconv"

	"github.com/cockroachdb/cockroachdb-parser/pkg/sql/lexbase"
)

// tokAt is the scanner token id for '@'.
const tokAt = int32('@')

// synthParam records a sqlc @name parameter discovered during preprocessing:
// the original parameter name (preserving source case) and the byte offset of
// the leading '@' in the original input.
type synthParam struct {
	name   string
	offset int
}

// rewriteAtParams replaces sqlc's @name parameter syntax with native CockroachDB
// $N placeholders so the query parses, and returns a map from placeholder index
// (0-based, matching tree.Placeholder.Idx) to the original parameter.
//
// CockroachDB overloads '@' (index hints like `table@index`, column ordinals
// like `@1`), so it cannot parse `@name` directly. We replace each `@name` with
// a `$N` placeholder padded with spaces to preserve byte length — keeping every
// other token at its original offset — and later reconstruct the A_Expr("@")
// node that sqlc's rewrite package already understands. Because the replacement
// is length-preserving, statement offsets computed against the rewritten text
// are valid against the original text the compiler edits.
func rewriteAtParams(src string) (string, map[int]synthParam) {
	toks := scanTokens(src)
	synth := map[int]synthParam{}
	if len(toks) == 0 {
		return src, synth
	}

	// Synthesized placeholders must not collide with the query's own $N
	// placeholders. CockroachDB preserves literal placeholder numbers per
	// statement, so a native $1 in one statement and a synthesized $1 in another
	// share the same index. Numbering synthesized placeholders above the largest
	// native one in the input keeps them distinct (a placeholder index present
	// in synth is always an @name; anything else is a real $N).
	maxNative := 0
	for _, t := range toks {
		if t.id == lexbase.PLACEHOLDER {
			if v, err := strconv.Atoi(t.str); err == nil && v > maxNative {
				maxNative = v
			}
		}
	}

	buf := []byte(src)
	num := maxNative
	for i := 0; i < len(toks); i++ {
		if toks[i].id != tokAt {
			continue
		}
		// The name must be an identifier immediately following '@' (no space).
		if i+1 >= len(toks) {
			continue
		}
		next := toks[i+1]
		if next.pos != toks[i].pos+1 {
			continue // e.g. "@ name" — not a parameter
		}
		if !isIdentStart(next.str) {
			continue // e.g. "@1" column ordinal, not a named parameter
		}
		// Exclude index hints like `table@index`, where '@' immediately follows
		// an identifier with no intervening whitespace.
		if i > 0 {
			prev := toks[i-1]
			if prev.pos+len(prev.str) == toks[i].pos && isIdentStart(prev.str) {
				continue
			}
		}

		atPos := toks[i].pos
		// The scanner folds identifier case but preserves length, so slice the
		// original text to recover the parameter's source spelling.
		nameLen := len(next.str)
		origName := src[next.pos : next.pos+nameLen]
		spanLen := 1 + nameLen // '@' + name

		num++
		repl := "$" + strconv.Itoa(num)
		if len(repl) > spanLen {
			// Cannot fit the placeholder in the span (a single-character name
			// with a large index). Leave it untouched; it will fail to parse,
			// surfacing a clear error rather than corrupting offsets.
			num--
			continue
		}

		for j := 0; j < spanLen; j++ {
			if j < len(repl) {
				buf[atPos+j] = repl[j]
			} else {
				buf[atPos+j] = ' '
			}
		}
		synth[num-1] = synthParam{name: origName, offset: atPos}
	}

	return string(buf), synth
}

// isIdentStart reports whether s begins with a character that can start a SQL
// identifier (a letter or underscore), distinguishing @name parameters from
// @1 column ordinals.
func isIdentStart(s string) bool {
	if s == "" {
		return false
	}
	c := s[0]
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}
