package cockroachdb

import (
	"strings"
	"testing"

	"github.com/rlpeck/crdb-sqlc/internal/sql/ast"
	"github.com/rlpeck/crdb-sqlc/internal/sql/astutils"
	"github.com/rlpeck/crdb-sqlc/internal/sql/named"
)

func parseOne(t *testing.T, sql string) *ast.RawStmt {
	t.Helper()
	stmts, err := (&Parser{}).Parse(strings.NewReader(sql))
	if err != nil {
		t.Fatalf("parse %q: %v", sql, err)
	}
	if len(stmts) != 1 {
		t.Fatalf("expected 1 statement, got %d for %q", len(stmts), sql)
	}
	return stmts[0].Raw
}

func TestUnresolvedNameOrdering(t *testing.T) {
	// schema.table.column must produce fields in significance order.
	raw := parseOne(t, "SELECT myschema.mytable.mycol FROM mytable")
	sel := raw.Stmt.(*ast.SelectStmt)
	res := sel.TargetList.Items[0].(*ast.ResTarget)
	ref := res.Val.(*ast.ColumnRef)
	var got []string
	for _, f := range ref.Fields.Items {
		s, ok := f.(*ast.String)
		if !ok {
			t.Fatalf("expected *ast.String field, got %T", f)
		}
		got = append(got, s.Str)
	}
	want := []string{"myschema", "mytable", "mycol"}
	if strings.Join(got, ".") != strings.Join(want, ".") {
		t.Fatalf("column ref fields = %v, want %v", got, want)
	}
}

func TestStarColumnRef(t *testing.T) {
	raw := parseOne(t, "SELECT * FROM foo")
	sel := raw.Stmt.(*ast.SelectStmt)
	res := sel.TargetList.Items[0].(*ast.ResTarget)
	ref := res.Val.(*ast.ColumnRef)
	if len(ref.Fields.Items) != 1 {
		t.Fatalf("expected 1 field, got %d", len(ref.Fields.Items))
	}
	if _, ok := ref.Fields.Items[0].(*ast.A_Star); !ok {
		t.Fatalf("expected *ast.A_Star, got %T", ref.Fields.Items[0])
	}
}

func TestSqlcArgFuncCall(t *testing.T) {
	// sqlc.arg(name) must convert to a FuncCall whose Func is recognized by the
	// named-parameter rewrite machinery, and must carry accurate offsets.
	sql := "SELECT id FROM foo WHERE name = sqlc.arg(target)"
	raw := parseOne(t, sql)
	sel := raw.Stmt.(*ast.SelectStmt)
	expr := sel.WhereClause.(*ast.A_Expr)
	call, ok := expr.Rexpr.(*ast.FuncCall)
	if !ok {
		t.Fatalf("expected *ast.FuncCall on rhs, got %T", expr.Rexpr)
	}
	if !named.IsParamFunc(call) {
		t.Fatalf("FuncCall not recognized as a sqlc param func: Func=%+v", call.Func)
	}
	if call.Func.Schema != "sqlc" || call.Func.Name != "arg" {
		t.Fatalf("Func = %+v, want {sqlc arg}", call.Func)
	}

	// Location must point at the literal "sqlc.arg(target)" in the source.
	wantLoc := strings.Index(sql, "sqlc.arg(target)")
	if call.Location != wantLoc {
		t.Fatalf("FuncCall.Location = %d, want %d", call.Location, wantLoc)
	}
	// The argument's position must point at "target" so origText is rebuilt
	// correctly by the rewrite package.
	argRef := call.Args.Items[0]
	wantArg := strings.Index(sql, "target")
	if argRef.Pos() != wantArg {
		t.Fatalf("arg Pos() = %d, want %d", argRef.Pos(), wantArg)
	}
}

func TestPlaceholder(t *testing.T) {
	raw := parseOne(t, "SELECT id FROM foo WHERE id = $1")
	sel := raw.Stmt.(*ast.SelectStmt)
	expr := sel.WhereClause.(*ast.A_Expr)
	ref, ok := expr.Rexpr.(*ast.ParamRef)
	if !ok {
		t.Fatalf("expected *ast.ParamRef, got %T", expr.Rexpr)
	}
	if ref.Number != 1 {
		t.Fatalf("ParamRef.Number = %d, want 1", ref.Number)
	}
	if !ref.Dollar {
		t.Fatalf("ParamRef.Dollar = false, want true")
	}
}

func TestCreateTable(t *testing.T) {
	raw := parseOne(t, "CREATE TABLE authors (id INT8 PRIMARY KEY, name STRING NOT NULL, bio STRING)")
	stmt := raw.Stmt.(*ast.CreateTableStmt)
	if stmt.Name.Name != "authors" {
		t.Fatalf("table name = %q, want authors", stmt.Name.Name)
	}
	if len(stmt.Cols) != 3 {
		t.Fatalf("expected 3 columns, got %d", len(stmt.Cols))
	}
	cases := []struct {
		name      string
		typ       string
		isNotNull bool
	}{
		{"id", "int8", true},
		{"name", "string", true},
		{"bio", "string", false},
	}
	for i, want := range cases {
		col := stmt.Cols[i]
		if col.Colname != want.name {
			t.Errorf("col %d name = %q, want %q", i, col.Colname, want.name)
		}
		if col.TypeName.Name != want.typ {
			t.Errorf("col %d type = %q, want %q", i, col.TypeName.Name, want.typ)
		}
		if col.IsNotNull != want.isNotNull {
			t.Errorf("col %d IsNotNull = %v, want %v", i, col.IsNotNull, want.isNotNull)
		}
	}
}

func TestArrayColumn(t *testing.T) {
	// Array columns must carry IsArray + ArrayDims (read by the catalog) and
	// ArrayBounds (read by the type-name path) so codegen emits []T, not T.
	raw := parseOne(t, "CREATE TABLE t (tags STRING[], scores INT8[], amounts DECIMAL(10,2)[])")
	stmt := raw.Stmt.(*ast.CreateTableStmt)
	if len(stmt.Cols) != 3 {
		t.Fatalf("expected 3 columns, got %d", len(stmt.Cols))
	}
	for _, col := range stmt.Cols {
		if !col.IsArray {
			t.Errorf("col %q: IsArray = false, want true", col.Colname)
		}
		if col.ArrayDims != 1 {
			t.Errorf("col %q: ArrayDims = %d, want 1", col.Colname, col.ArrayDims)
		}
		if col.TypeName.ArrayBounds == nil || len(col.TypeName.ArrayBounds.Items) != 1 {
			t.Errorf("col %q: ArrayBounds not populated", col.Colname)
		}
	}

	// A scalar column must not be flagged as an array.
	col := parseOne(t, "CREATE TABLE t (name STRING)").Stmt.(*ast.CreateTableStmt).Cols[0]
	if col.IsArray || col.ArrayDims != 0 {
		t.Errorf("scalar column flagged as array: IsArray=%v ArrayDims=%d", col.IsArray, col.ArrayDims)
	}
}

func TestArrayCastParam(t *testing.T) {
	// An array cast (e.g. @ids::int8[]) must carry ArrayBounds so the parameter
	// resolves to []T, not T.
	raw := parseOne(t, "SELECT id FROM t WHERE tags = @vals::int8[]")
	cast, ok := raw.Stmt.(*ast.SelectStmt).WhereClause.(*ast.A_Expr).Rexpr.(*ast.TypeCast)
	if !ok {
		t.Fatalf("expected *ast.TypeCast on rhs, got %T", raw.Stmt.(*ast.SelectStmt).WhereClause.(*ast.A_Expr).Rexpr)
	}
	if cast.TypeName.Name != "int8" {
		t.Errorf("cast type name = %q, want int8", cast.TypeName.Name)
	}
	if cast.TypeName.ArrayBounds == nil || len(cast.TypeName.ArrayBounds.Items) != 1 {
		t.Fatalf("cast TypeName.ArrayBounds not set for int8[]")
	}

	// A scalar cast must not gain array bounds.
	c2 := parseOne(t, "SELECT id FROM t WHERE x = @v::int8").Stmt.(*ast.SelectStmt).WhereClause.(*ast.A_Expr).Rexpr.(*ast.TypeCast)
	if c2.TypeName.ArrayBounds != nil {
		t.Errorf("scalar cast wrongly has ArrayBounds")
	}
}

func TestInsertUpdateDelete(t *testing.T) {
	if _, ok := parseOne(t, "INSERT INTO foo (id, name) VALUES ($1, $2)").Stmt.(*ast.InsertStmt); !ok {
		t.Fatal("expected *ast.InsertStmt")
	}
	if _, ok := parseOne(t, "UPDATE foo SET name = $1 WHERE id = $2").Stmt.(*ast.UpdateStmt); !ok {
		t.Fatal("expected *ast.UpdateStmt")
	}
	if _, ok := parseOne(t, "DELETE FROM foo WHERE id = $1").Stmt.(*ast.DeleteStmt); !ok {
		t.Fatal("expected *ast.DeleteStmt")
	}
}

func TestJoin(t *testing.T) {
	raw := parseOne(t, "SELECT a.id FROM a JOIN b ON a.id = b.a_id")
	sel := raw.Stmt.(*ast.SelectStmt)
	if len(sel.FromClause.Items) != 1 {
		t.Fatalf("expected 1 from item, got %d", len(sel.FromClause.Items))
	}
	join, ok := sel.FromClause.Items[0].(*ast.JoinExpr)
	if !ok {
		t.Fatalf("expected *ast.JoinExpr, got %T", sel.FromClause.Items[0])
	}
	if join.Jointype != ast.JoinTypeInner {
		t.Fatalf("join type = %d, want inner", join.Jointype)
	}
	if join.Quals == nil {
		t.Fatal("expected join qualifiers")
	}
}

func TestAtNameParam(t *testing.T) {
	// @name is rewritten to the A_Expr("@") node the rewrite package recognizes,
	// with a Location pointing at the literal '@' in the source.
	sql := "SELECT id FROM foo WHERE name = @target"
	raw := parseOne(t, sql)
	sel := raw.Stmt.(*ast.SelectStmt)
	cmp := sel.WhereClause.(*ast.A_Expr) // name = @target
	at, ok := cmp.Rexpr.(*ast.A_Expr)
	if !ok {
		t.Fatalf("expected @name A_Expr on rhs, got %T", cmp.Rexpr)
	}
	if !named.IsParamSign(at) {
		t.Fatalf("@name node not recognized as a param sign: Name=%+v", at.Name)
	}
	if want := strings.Index(sql, "@target"); at.Location != want {
		t.Fatalf("@name Location = %d, want %d", at.Location, want)
	}
	if name, _ := flattenString(at.Rexpr); name != "target" {
		t.Fatalf("@name param name = %q, want target", name)
	}
}

func TestAtNameInCaseAndCast(t *testing.T) {
	// A parameter buried in a CASE and behind a cast must still be converted
	// (these previously leaked through as a literal @name).
	raw := parseOne(t, "SELECT id FROM foo WHERE (CASE WHEN @flag::boolean THEN name = @target ELSE true END)")
	if n := countParamSigns(raw); n != 2 {
		t.Fatalf("expected 2 @name params inside CASE, found %d", n)
	}
}

func TestAtNameInOnConflict(t *testing.T) {
	raw := parseOne(t, "INSERT INTO foo (id, name) VALUES (@id, @name) ON CONFLICT (id) DO UPDATE SET name = @name")
	// @id, and @name appearing twice (deduped by name later) -> 3 sign nodes.
	if n := countParamSigns(raw); n != 3 {
		t.Fatalf("expected 3 @name param nodes, found %d", n)
	}
}

func TestInsertValuesHasNonNilLists(t *testing.T) {
	// The embedded VALUES SelectStmt must have non-nil TargetList/FromClause/
	// ValuesLists, otherwise the compiler's walkers nil-deref.
	raw := parseOne(t, "INSERT INTO foo (id) VALUES ($1) RETURNING *")
	ins := raw.Stmt.(*ast.InsertStmt)
	sel, ok := ins.SelectStmt.(*ast.SelectStmt)
	if !ok {
		t.Fatalf("expected embedded *ast.SelectStmt, got %T", ins.SelectStmt)
	}
	if sel.TargetList == nil || sel.FromClause == nil || sel.ValuesLists == nil {
		t.Fatalf("VALUES SelectStmt has a nil list: target=%v from=%v values=%v",
			sel.TargetList == nil, sel.FromClause == nil, sel.ValuesLists == nil)
	}
}

func flattenString(n ast.Node) (string, bool) {
	if ref, ok := n.(*ast.ColumnRef); ok {
		var out string
		for _, f := range ref.Fields.Items {
			if s, ok := f.(*ast.String); ok {
				out += s.Str
			}
		}
		return out, true
	}
	return "", false
}

func countParamSigns(raw *ast.RawStmt) int {
	n := 0
	astutils.Walk(astutils.VisitorFunc(func(node ast.Node) {
		if named.IsParamSign(node) {
			n++
		}
	}), raw)
	return n
}

func TestReservedKeyword(t *testing.T) {
	p := &Parser{}
	if !p.IsReservedKeyword("select") {
		t.Error("select should be reserved")
	}
	if p.IsReservedKeyword("name") {
		t.Error("name should not be reserved")
	}
}
