package cockroachdb

import (
	"go/constant"
	"strings"

	"github.com/cockroachdb/cockroachdb-parser/pkg/sql/sem/tree"

	"github.com/rlpeck/crdb-sqlc/internal/sql/ast"
)

// cc converts a cockroachdb-parser tree.Statement into sqlc's PostgreSQL-shaped
// internal AST. Unsupported nodes become *ast.TODO, which the parser skips at
// the statement level and downstream walkers tolerate at the expression level.
type cc struct {
	loc *locator
	// synth maps a native placeholder index back to the sqlc @name parameter it
	// stands in for (see rewriteAtParams). nil/absent means a real $N placeholder.
	synth map[int]synthParam
}

func (c *cc) convert(node tree.Statement) ast.Node {
	switch n := node.(type) {
	case *tree.Select:
		return c.convertSelect(n)
	case *tree.SelectClause:
		return c.convertSelectStatement(n)
	case *tree.Insert:
		return c.convertInsert(n)
	case *tree.Update:
		return c.convertUpdate(n)
	case *tree.Delete:
		return c.convertDelete(n)
	case *tree.CreateTable:
		return c.convertCreateTable(n)
	default:
		return todo("convert", node)
	}
}

// --- SELECT --------------------------------------------------------------

func (c *cc) convertSelect(n *tree.Select) ast.Node {
	inner := c.convertSelectStatement(n.Select)
	stmt, ok := inner.(*ast.SelectStmt)
	if !ok {
		return inner
	}
	if n.Limit != nil {
		if n.Limit.Count != nil {
			stmt.LimitCount = c.convertExpr(n.Limit.Count)
		}
		if n.Limit.Offset != nil {
			stmt.LimitOffset = c.convertExpr(n.Limit.Offset)
		}
	}
	return stmt
}

func (c *cc) convertSelectStatement(node tree.SelectStatement) ast.Node {
	switch n := node.(type) {
	case *tree.SelectClause:
		return c.convertSelectClause(n)
	case *tree.ParenSelect:
		return c.convertSelect(n.Select)
	case *tree.ValuesClause:
		return c.convertValues(n)
	default:
		return todo("convertSelectStatement", node)
	}
}

func (c *cc) convertSelectClause(n *tree.SelectClause) ast.Node {
	stmt := &ast.SelectStmt{
		TargetList: c.convertSelectExprs(tree.SelectExprs(n.Exprs)),
		FromClause: c.convertFrom(n.From),
		// ValuesLists is read unconditionally when this statement is an INSERT's
		// source, so it must never be nil.
		ValuesLists: &ast.List{},
	}
	if n.Where != nil {
		stmt.WhereClause = c.convertExpr(n.Where.Expr)
	}
	if n.Having != nil {
		stmt.HavingClause = c.convertExpr(n.Having.Expr)
	}
	if len(n.GroupBy) > 0 {
		group := &ast.List{}
		for _, e := range n.GroupBy {
			group.Items = append(group.Items, c.convertExpr(e))
		}
		stmt.GroupClause = group
	}
	if n.Distinct {
		stmt.DistinctClause = &ast.List{}
	}
	return stmt
}

func (c *cc) convertValues(n *tree.ValuesClause) ast.Node {
	// TargetList, FromClause, and ValuesLists are all read unconditionally by
	// the compiler (e.g. find_params and sourceTables walk them), so none may be
	// nil — a VALUES clause carries empty lists for the first two.
	stmt := &ast.SelectStmt{
		TargetList:  &ast.List{},
		FromClause:  &ast.List{},
		ValuesLists: &ast.List{},
	}
	for _, row := range n.Rows {
		rowList := &ast.List{}
		for _, e := range row {
			rowList.Items = append(rowList.Items, c.convertExpr(e))
		}
		stmt.ValuesLists.Items = append(stmt.ValuesLists.Items, rowList)
	}
	return stmt
}

func (c *cc) convertSelectExprs(exprs tree.SelectExprs) *ast.List {
	list := &ast.List{}
	for _, e := range exprs {
		list.Items = append(list.Items, c.convertSelectExpr(e))
	}
	return list
}

func (c *cc) convertSelectExpr(e tree.SelectExpr) ast.Node {
	res := &ast.ResTarget{}
	if e.As != "" {
		name := string(e.As)
		res.Name = &name
	}
	switch expr := e.Expr.(type) {
	case tree.UnqualifiedStar:
		loc := c.bareStar()
		res.Location = loc
		res.Val = &ast.ColumnRef{
			Fields:   &ast.List{Items: []ast.Node{&ast.A_Star{}}},
			Location: loc,
		}
	case *tree.AllColumnsSelector:
		table := ""
		if expr.TableName != nil {
			table = expr.TableName.Object()
		}
		loc := 0
		if c.loc != nil {
			if p, ok := c.loc.qualifiedStar(table); ok {
				loc = p
			}
		}
		res.Location = loc
		var items []ast.Node
		if table != "" {
			items = append(items, NewIdentifier(table))
		}
		items = append(items, &ast.A_Star{})
		res.Val = &ast.ColumnRef{Fields: &ast.List{Items: items}, Location: loc}
	default:
		res.Val = c.convertExpr(e.Expr)
	}
	return res
}

func (c *cc) bareStar() int {
	if c.loc == nil {
		return 0
	}
	if p, ok := c.loc.popBareStar(); ok {
		return p
	}
	return 0
}

// --- FROM / tables -------------------------------------------------------

func (c *cc) convertFrom(from tree.From) *ast.List {
	list := &ast.List{}
	for _, te := range from.Tables {
		list.Items = append(list.Items, c.convertTableExpr(te))
	}
	return list
}

func (c *cc) convertTableExpr(te tree.TableExpr) ast.Node {
	switch n := te.(type) {
	case *tree.AliasedTableExpr:
		return c.convertAliasedTableExpr(n)
	case *tree.JoinTableExpr:
		return c.convertJoin(n)
	case *tree.ParenTableExpr:
		return c.convertTableExpr(n.Expr)
	default:
		return todo("convertTableExpr", te)
	}
}

func (c *cc) convertAliasedTableExpr(n *tree.AliasedTableExpr) ast.Node {
	switch e := n.Expr.(type) {
	case *tree.TableName:
		rv := c.tableNameToRangeVar(e)
		c.applyAlias(rv, n.As)
		return rv
	default:
		return todo("convertAliasedTableExpr", n.Expr)
	}
}

func (c *cc) applyAlias(rv *ast.RangeVar, as tree.AliasClause) {
	if rv == nil || as.Alias == "" {
		return
	}
	name := string(as.Alias)
	rv.Alias = &ast.Alias{Aliasname: &name}
}

func (c *cc) convertJoin(j *tree.JoinTableExpr) ast.Node {
	je := &ast.JoinExpr{
		Jointype: joinType(j.JoinType),
		Larg:     c.convertTableExpr(j.Left),
		Rarg:     c.convertTableExpr(j.Right),
	}
	switch cond := j.Cond.(type) {
	case *tree.OnJoinCond:
		je.Quals = c.convertExpr(cond.Expr)
	case *tree.UsingJoinCond:
		using := &ast.List{}
		for _, col := range cond.Cols {
			using.Items = append(using.Items, NewIdentifier(string(col)))
		}
		je.UsingClause = using
	case tree.NaturalJoinCond:
		je.IsNatural = true
	}
	return je
}

func joinType(s string) ast.JoinType {
	switch strings.ToUpper(s) {
	case "LEFT":
		return ast.JoinTypeLeft
	case "RIGHT":
		return ast.JoinTypeRight
	case "FULL":
		return ast.JoinTypeFull
	default:
		// "", INNER, CROSS
		return ast.JoinTypeInner
	}
}

func (c *cc) tableNameToRangeVar(tn *tree.TableName) *ast.RangeVar {
	rel := tn.Table()
	rv := &ast.RangeVar{Relname: &rel}
	if tn.ExplicitSchema {
		schema := tn.Schema()
		rv.Schemaname = &schema
	}
	if tn.ExplicitCatalog {
		cat := tn.Catalog()
		rv.Catalogname = &cat
	}
	return rv
}

func (c *cc) tableExprToRangeVar(te tree.TableExpr) *ast.RangeVar {
	switch n := te.(type) {
	case *tree.AliasedTableExpr:
		if tn, ok := n.Expr.(*tree.TableName); ok {
			rv := c.tableNameToRangeVar(tn)
			c.applyAlias(rv, n.As)
			return rv
		}
	case *tree.TableName:
		return c.tableNameToRangeVar(n)
	}
	return nil
}

func (c *cc) tableNameToTableName(tn *tree.TableName) *ast.TableName {
	out := &ast.TableName{Name: tn.Table()}
	if tn.ExplicitSchema {
		out.Schema = tn.Schema()
	}
	if tn.ExplicitCatalog {
		out.Catalog = tn.Catalog()
	}
	return out
}

// --- INSERT / UPDATE / DELETE -------------------------------------------

func (c *cc) convertInsert(n *tree.Insert) ast.Node {
	stmt := &ast.InsertStmt{
		Relation:         c.tableExprToRangeVar(n.Table),
		Cols:             c.nameListToResTargets(n.Columns),
		ReturningList:    c.convertReturning(n.Returning),
		OnConflictClause: c.convertOnConflict(n.OnConflict),
	}
	if n.Rows != nil {
		stmt.SelectStmt = c.convertSelect(n.Rows)
	}
	return stmt
}

// CockroachDB ON CONFLICT actions. Only carried for completeness; the value is
// not consumed by codegen (the query text is emitted verbatim), but the DO
// UPDATE SET expressions must be converted so their @name parameters are found.
const (
	onConflictNothing ast.OnConflictAction = 1
	onConflictUpdate  ast.OnConflictAction = 2
)

func (c *cc) convertOnConflict(oc *tree.OnConflict) *ast.OnConflictClause {
	if oc == nil {
		return nil
	}
	clause := &ast.OnConflictClause{TargetList: &ast.List{}}
	if oc.DoNothing {
		clause.Action = onConflictNothing
		return clause
	}
	clause.Action = onConflictUpdate
	for _, ue := range oc.Exprs {
		for _, nm := range ue.Names {
			name := string(nm)
			clause.TargetList.Items = append(clause.TargetList.Items, &ast.ResTarget{
				Name: &name,
				Val:  c.convertExpr(ue.Expr),
			})
		}
	}
	if oc.Where != nil {
		clause.WhereClause = c.convertExpr(oc.Where.Expr)
	}
	return clause
}

func (c *cc) convertUpdate(n *tree.Update) ast.Node {
	stmt := &ast.UpdateStmt{
		Relations:     &ast.List{Items: []ast.Node{c.tableExprToRangeVar(n.Table)}},
		TargetList:    &ast.List{},
		FromClause:    &ast.List{},
		ReturningList: c.convertReturning(n.Returning),
	}
	for _, ue := range n.Exprs {
		for _, nm := range ue.Names {
			name := string(nm)
			stmt.TargetList.Items = append(stmt.TargetList.Items, &ast.ResTarget{
				Name: &name,
				Val:  c.convertExpr(ue.Expr),
			})
		}
	}
	if n.Where != nil {
		stmt.WhereClause = c.convertExpr(n.Where.Expr)
	}
	return stmt
}

func (c *cc) convertDelete(n *tree.Delete) ast.Node {
	stmt := &ast.DeleteStmt{
		Relations:     &ast.List{Items: []ast.Node{c.tableExprToRangeVar(n.Table)}},
		ReturningList: c.convertReturning(n.Returning),
	}
	if n.Where != nil {
		stmt.WhereClause = c.convertExpr(n.Where.Expr)
	}
	return stmt
}

func (c *cc) nameListToResTargets(names tree.NameList) *ast.List {
	list := &ast.List{}
	for _, nm := range names {
		name := string(nm)
		list.Items = append(list.Items, &ast.ResTarget{Name: &name})
	}
	return list
}

func (c *cc) convertReturning(r tree.ReturningClause) *ast.List {
	if exprs, ok := r.(*tree.ReturningExprs); ok {
		return c.convertSelectExprs(tree.SelectExprs(*exprs))
	}
	// outputColumns reads ReturningList unconditionally, so never return nil.
	return &ast.List{}
}

// --- CREATE TABLE --------------------------------------------------------

func (c *cc) convertCreateTable(n *tree.CreateTable) ast.Node {
	stmt := &ast.CreateTableStmt{
		Name:        c.tableNameToTableName(&n.Table),
		IfNotExists: n.IfNotExists,
	}
	for _, def := range n.Defs {
		col, ok := def.(*tree.ColumnTableDef)
		if !ok {
			continue
		}
		typeName, dims := convertTypeRef(col.Type)
		stmt.Cols = append(stmt.Cols, &ast.ColumnDef{
			Colname:    string(col.Name),
			TypeName:   typeName,
			IsNotNull:  col.Nullable.Nullability == tree.NotNull || col.PrimaryKey.IsPrimaryKey,
			IsArray:    dims > 0,
			ArrayDims:  dims,
			PrimaryKey: col.PrimaryKey.IsPrimaryKey,
		})
	}
	return stmt
}

func typeRefString(ref tree.ResolvableTypeReference) string {
	if ref == nil {
		return ""
	}
	return ref.SQLString()
}

// convertTypeRef builds an ast.TypeName from a CockroachDB type reference,
// returning the array dimension count (0 for scalars). Array types set
// ArrayBounds so that both the catalog (defineColumn) and the type-name path
// (to_column.go) treat the value as []T — this matters for array columns *and*
// array casts like @ids::int8[].
func convertTypeRef(ref tree.ResolvableTypeReference) (*ast.TypeName, int) {
	name, isArray := normalizeTypeName(typeRefString(ref))
	tn := &ast.TypeName{Name: name}
	if isArray {
		// CockroachDB arrays are single-dimensional.
		tn.ArrayBounds = &ast.List{Items: []ast.Node{&ast.Integer{Ival: -1}}}
		return tn, 1
	}
	return tn, 0
}

// normalizeTypeName reduces a CockroachDB type spelling to a canonical base
// name and reports whether it is an array. e.g. "DECIMAL(10,2)[]" -> "decimal", true.
func normalizeTypeName(s string) (string, bool) {
	s = strings.TrimSpace(s)
	isArray := false
	if strings.HasSuffix(s, "[]") {
		isArray = true
		s = strings.TrimSuffix(s, "[]")
	}
	if i := strings.IndexByte(s, '('); i >= 0 {
		s = s[:i]
	}
	return strings.ToLower(strings.TrimSpace(s)), isArray
}

// --- expressions ---------------------------------------------------------

func (c *cc) convertExpr(node tree.Expr) ast.Node {
	if node == nil {
		return nil
	}
	if node == tree.DNull {
		return &ast.A_Const{Val: &ast.Null{}}
	}
	switch n := node.(type) {
	case *tree.UnresolvedName:
		return c.convertUnresolvedName(n)
	case *tree.ColumnItem:
		return c.convertColumnItem(n)
	case tree.UnqualifiedStar:
		return &ast.ColumnRef{Fields: &ast.List{Items: []ast.Node{&ast.A_Star{}}}}
	case *tree.AllColumnsSelector:
		return c.convertAllColumnsExpr(n)
	case *tree.NumVal:
		return c.convertNumVal(n)
	case *tree.StrVal:
		return &ast.A_Const{Val: &ast.String{Str: n.RawString()}}
	case *tree.DBool:
		return &ast.A_Const{Val: &ast.Boolean{Boolval: bool(*n)}}
	case *tree.Placeholder:
		// A placeholder that stands in for a sqlc @name parameter becomes the
		// A_Expr("@") node the rewrite package recognizes (matching the
		// PostgreSQL engine); a genuine $N placeholder becomes a ParamRef.
		if sp, ok := c.synth[int(n.Idx)]; ok {
			return &ast.A_Expr{
				Kind: ast.A_Expr_Kind_OP,
				Name: &ast.List{Items: []ast.Node{&ast.String{Str: "@"}}},
				Rexpr: &ast.ColumnRef{
					Fields:   &ast.List{Items: []ast.Node{&ast.String{Str: sp.name}}},
					Location: sp.offset + 1, // points just past '@', at the name
				},
				Location: sp.offset,
			}
		}
		return &ast.ParamRef{Number: int(n.Idx) + 1, Dollar: true}
	case *tree.FuncExpr:
		return c.convertFuncExpr(n)
	case *tree.ComparisonExpr:
		return c.convertComparison(n)
	case *tree.BinaryExpr:
		return c.convertBinary(n)
	case *tree.AndExpr:
		return c.boolExpr(ast.BoolExprTypeAnd, c.convertExpr(n.Left), c.convertExpr(n.Right))
	case *tree.OrExpr:
		return c.boolExpr(ast.BoolExprTypeOr, c.convertExpr(n.Left), c.convertExpr(n.Right))
	case *tree.NotExpr:
		return c.boolExpr(ast.BoolExprTypeNot, c.convertExpr(n.Expr))
	case *tree.ParenExpr:
		return c.convertExpr(n.Expr)
	case *tree.CaseExpr:
		return c.convertCaseExpr(n)
	case *tree.CoalesceExpr:
		// COALESCE/IFNULL: type (and any parameter's type) comes from the
		// arguments, so use CoalesceExpr for sibling-based inference.
		return &ast.CoalesceExpr{Args: c.convertExprList(n.Exprs)}
	case *tree.CastExpr:
		typeName, _ := convertTypeRef(n.Type)
		return &ast.TypeCast{
			Arg:      c.convertExpr(n.Expr),
			TypeName: typeName,
		}
	default:
		return todo("convertExpr", node)
	}
}

func (c *cc) boolExpr(op ast.BoolExprType, args ...ast.Node) ast.Node {
	return &ast.BoolExpr{
		Boolop: op,
		Args:   &ast.List{Items: args},
	}
}

func (c *cc) convertUnresolvedName(n *tree.UnresolvedName) *ast.ColumnRef {
	var items []ast.Node
	end := 0
	if n.Star {
		// Parts[0] is the (empty) column slot for `tbl.*`.
		end = 1
	}
	// Parts are stored most-significant-last; emit schema..column order.
	for i := n.NumParts - 1; i >= end; i-- {
		items = append(items, NewIdentifier(n.Parts[i]))
	}
	if n.Star {
		items = append(items, &ast.A_Star{})
	}
	return &ast.ColumnRef{Fields: &ast.List{Items: items}}
}

func (c *cc) convertColumnItem(n *tree.ColumnItem) *ast.ColumnRef {
	var items []ast.Node
	if n.TableName != nil {
		if n.TableName.NumParts >= 3 {
			items = append(items, NewIdentifier(n.TableName.Catalog()))
		}
		if n.TableName.NumParts >= 2 {
			items = append(items, NewIdentifier(n.TableName.Schema()))
		}
		items = append(items, NewIdentifier(n.TableName.Object()))
	}
	items = append(items, NewIdentifier(string(n.ColumnName)))
	return &ast.ColumnRef{Fields: &ast.List{Items: items}}
}

func (c *cc) convertAllColumnsExpr(n *tree.AllColumnsSelector) *ast.ColumnRef {
	var items []ast.Node
	if n.TableName != nil {
		items = append(items, NewIdentifier(n.TableName.Object()))
	}
	items = append(items, &ast.A_Star{})
	return &ast.ColumnRef{Fields: &ast.List{Items: items}}
}

func (c *cc) convertCaseExpr(n *tree.CaseExpr) ast.Node {
	ce := &ast.CaseExpr{Args: &ast.List{}}
	if n.Expr != nil {
		ce.Arg = c.convertExpr(n.Expr)
	}
	for _, w := range n.Whens {
		ce.Args.Items = append(ce.Args.Items, &ast.CaseWhen{
			Expr:   c.convertExpr(w.Cond),
			Result: c.convertExpr(w.Val),
		})
	}
	if n.Else != nil {
		ce.Defresult = c.convertExpr(n.Else)
	}
	return ce
}

func (c *cc) convertNumVal(n *tree.NumVal) ast.Node {
	if n.Kind() == constant.Int {
		if i, err := n.AsInt64(); err == nil {
			return &ast.A_Const{Val: &ast.Integer{Ival: i}}
		}
	}
	s := n.OrigString()
	if s == "" {
		s = n.String()
	}
	return &ast.A_Const{Val: &ast.Float{Str: s}}
}

func (c *cc) convertComparison(n *tree.ComparisonExpr) ast.Node {
	op := strings.TrimSpace(n.Operator.String())
	kind := ast.A_Expr_Kind_OP
	switch strings.ToUpper(op) {
	case "IN", "NOT IN":
		kind = ast.A_Expr_Kind_IN
	case "LIKE", "NOT LIKE":
		kind = ast.A_Expr_Kind_LIKE
	case "ILIKE", "NOT ILIKE":
		kind = ast.A_Expr_Kind_ILIKE
	}
	return &ast.A_Expr{
		Kind:  kind,
		Name:  &ast.List{Items: []ast.Node{&ast.String{Str: op}}},
		Lexpr: c.convertExpr(n.Left),
		Rexpr: c.convertExpr(n.Right),
	}
}

func (c *cc) convertBinary(n *tree.BinaryExpr) ast.Node {
	return &ast.A_Expr{
		Kind:  ast.A_Expr_Kind_OP,
		Name:  &ast.List{Items: []ast.Node{&ast.String{Str: strings.TrimSpace(n.Operator.String())}}},
		Lexpr: c.convertExpr(n.Left),
		Rexpr: c.convertExpr(n.Right),
	}
}

func (c *cc) convertFuncExpr(n *tree.FuncExpr) ast.Node {
	schema, name := c.funcName(n)

	// sqlc.arg()/narg()/slice() need accurate source offsets so the rewrite
	// package can replace them with $N placeholders.
	if schema == "sqlc" && isSqlcParamFunc(name) && len(n.Exprs) == 1 {
		return c.convertSqlcParam(n, name)
	}

	// GREATEST/LEAST derive their type (and any parameter's type) from their
	// arguments, not a fixed catalog signature. Convert them to MinMaxExpr so a
	// parameter like GREATEST(col, @p) infers col's type instead of falling back
	// to interface{}. (COALESCE is handled in convertExpr — CockroachDB parses
	// it as its own node, not a FuncExpr.)
	if schema == "" {
		switch name {
		case "greatest", "least":
			op := ast.MinMaxOp(0) // IS_GREATEST
			if name == "least" {
				op = ast.MinMaxOp(1) // IS_LEAST
			}
			return &ast.MinMaxExpr{Op: op, Args: c.convertExprList(n.Exprs)}
		}
	}

	items := []ast.Node{}
	if schema != "" {
		items = append(items, NewIdentifier(schema))
	}
	items = append(items, NewIdentifier(name))

	args := &ast.List{}
	aggStar := false
	for _, e := range n.Exprs {
		if _, ok := e.(tree.UnqualifiedStar); ok {
			aggStar = true
			continue
		}
		args.Items = append(args.Items, c.convertExpr(e))
	}

	return &ast.FuncCall{
		Func:     &ast.FuncName{Schema: schema, Name: name},
		Funcname: &ast.List{Items: items},
		Args:     args,
		AggStar:  aggStar,
	}
}

func (c *cc) convertExprList(exprs tree.Exprs) *ast.List {
	list := &ast.List{}
	for _, e := range exprs {
		list.Items = append(list.Items, c.convertExpr(e))
	}
	return list
}

func (c *cc) funcName(n *tree.FuncExpr) (schema, name string) {
	switch ref := n.Func.FunctionReference.(type) {
	case *tree.UnresolvedName:
		name = identifier(ref.Parts[0])
		if ref.NumParts >= 2 {
			schema = identifier(ref.Parts[1])
		}
	case *tree.FunctionDefinition:
		name = identifier(ref.Name)
	case *tree.ResolvedFunctionDefinition:
		name = identifier(ref.Name)
	}
	return schema, name
}

func (c *cc) convertSqlcParam(n *tree.FuncExpr, kind string) ast.Node {
	argName := ""
	if un, ok := n.Exprs[0].(*tree.UnresolvedName); ok && un.NumParts >= 1 {
		argName = un.Parts[0]
	}

	funcLoc, argLoc := 0, 0
	if c.loc != nil {
		if pl, ok := c.loc.popParam(kind, argName); ok {
			funcLoc = pl.funcAbs
			argLoc = pl.argAbs
		}
	}

	arg := &ast.ColumnRef{
		Fields:   &ast.List{Items: []ast.Node{&ast.String{Str: argName}}},
		Location: argLoc,
	}
	return &ast.FuncCall{
		Func:     &ast.FuncName{Schema: "sqlc", Name: kind},
		Funcname: &ast.List{Items: []ast.Node{NewIdentifier("sqlc"), NewIdentifier(kind)}},
		Args:     &ast.List{Items: []ast.Node{arg}},
		Location: funcLoc,
	}
}
