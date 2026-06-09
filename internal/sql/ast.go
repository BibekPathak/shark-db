package sql

import "fmt"

type Node interface {
	String() string
}

type SelectStmt struct {
	Columns []string
	Table   string
	Where   Expr
	OrderBy []OrderByCol
	Limit   int
}

func (s *SelectStmt) String() string {
	return fmt.Sprintf("SELECT %v FROM %s", s.Columns, s.Table)
}

type InsertStmt struct {
	Table string
	Key   string
	Value string
}

func (i *InsertStmt) String() string {
	return fmt.Sprintf("INSERT INTO %s VALUES (%s, %s)", i.Table, i.Key, i.Value)
}

type UpdateStmt struct {
	Table      string
	SetClauses []SetClause
	Where      Expr
}

func (u *UpdateStmt) String() string {
	return fmt.Sprintf("UPDATE %s SET %v", u.Table, u.SetClauses)
}

type DeleteStmt struct {
	Table string
	Where Expr
}

func (d *DeleteStmt) String() string {
	return fmt.Sprintf("DELETE FROM %s", d.Table)
}

type CreateTableStmt struct {
	Table   string
	Columns []string
	Indexes []string
}

func (c *CreateTableStmt) String() string {
	return fmt.Sprintf("CREATE TABLE %s (%v)", c.Table, c.Columns)
}

type CreateIndexStmt struct {
	Table  string
	Column string
}

func (c *CreateIndexStmt) String() string {
	return fmt.Sprintf("CREATE INDEX ON %s (%s)", c.Table, c.Column)
}

type DropTableStmt struct {
	Table string
}

func (d *DropTableStmt) String() string {
	return fmt.Sprintf("DROP TABLE %s", d.Table)
}

type DropIndexStmt struct {
	Table  string
	Column string
}

func (d *DropIndexStmt) String() string {
	return fmt.Sprintf("DROP INDEX %s (%s)", d.Table, d.Column)
}

type SetClause struct {
	Column string
	Value  Expr
}

func (s *SetClause) String() string {
	return fmt.Sprintf("%s=%v", s.Column, s.Value)
}

type OrderByCol struct {
	Column string
	Desc   bool
}

func (o *OrderByCol) String() string {
	if o.Desc {
		return o.Column + " DESC"
	}
	return o.Column + " ASC"
}

type Expr interface {
	String() string
}

type ComparisonExpr struct {
	Op    string
	Left  Expr
	Right Expr
}

func (c *ComparisonExpr) String() string {
	if c.Op == "IS NULL" || c.Op == "IS NOT NULL" {
		return fmt.Sprintf("%v %s", c.Left, c.Op)
	}
	return fmt.Sprintf("%v %s %v", c.Left, c.Op, c.Right)
}

type LogicalExpr struct {
	Op    string
	Left  Expr
	Right Expr
}

func (l *LogicalExpr) String() string {
	if l.Op == "NOT" {
		return fmt.Sprintf("NOT %v", l.Left)
	}
	return fmt.Sprintf("%v %s %v", l.Left, l.Op, l.Right)
}

type ColumnRef struct {
	Column string
}

func (c *ColumnRef) String() string {
	return c.Column
}

type Literal struct {
	Value interface{}
}

func (l *Literal) String() string {
	return fmt.Sprintf("%v", l.Value)
}

type ListExpr struct {
	Exprs []Expr
}

func (l *ListExpr) String() string {
	strs := make([]string, len(l.Exprs))
	for i, e := range l.Exprs {
		strs[i] = e.String()
	}
	return fmt.Sprintf("(%s)", joinStrings(strs, ", "))
}

type ParamRef struct {
	Index int
}

func (p *ParamRef) String() string {
	return fmt.Sprintf("$%d", p.Index)
}

func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}
