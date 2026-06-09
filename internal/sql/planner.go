package sql

import "fmt"

type PlanNode interface {
	Execute(engine interface{}) ([][]string, error)
	String() string
}

type Planner struct {
	engine interface{}
}

func NewPlanner(engine interface{}) *Planner {
	return &Planner{engine: engine}
}

func (p *Planner) BuildPlan(stmt Node) (PlanNode, error) {
	switch s := stmt.(type) {
	case *SelectStmt:
		return p.buildSelectPlan(s)
	case *InsertStmt:
		return p.buildInsertPlan(s)
	case *UpdateStmt:
		return p.buildUpdatePlan(s)
	case *DeleteStmt:
		return p.buildDeletePlan(s)
	case *CreateTableStmt:
		return p.buildCreateTablePlan(s)
	case *CreateIndexStmt:
		return p.buildCreateIndexPlan(s)
	case *DropTableStmt:
		return p.buildDropTablePlan(s)
	case *DropIndexStmt:
		return p.buildDropIndexPlan(s)
	default:
		return nil, fmt.Errorf("unsupported statement type: %T", stmt)
	}
}

func (p *Planner) buildSelectPlan(stmt *SelectStmt) (PlanNode, error) {
	return &TableScanPlan{
		Table:   stmt.Table,
		Columns: stmt.Columns,
		Filter:  stmt.Where,
		OrderBy: stmt.OrderBy,
		Limit:   stmt.Limit,
	}, nil
}

func (p *Planner) buildInsertPlan(stmt *InsertStmt) (PlanNode, error) {
	return &InsertPlan{
		Table: stmt.Table,
		Key:   stmt.Key,
		Value: stmt.Value,
	}, nil
}

func (p *Planner) buildUpdatePlan(stmt *UpdateStmt) (PlanNode, error) {
	return &UpdatePlan{
		Table:      stmt.Table,
		SetClauses: stmt.SetClauses,
		Where:      stmt.Where,
	}, nil
}

func (p *Planner) buildDeletePlan(stmt *DeleteStmt) (PlanNode, error) {
	return &DeletePlan{
		Table: stmt.Table,
		Where: stmt.Where,
	}, nil
}

func (p *Planner) buildCreateTablePlan(stmt *CreateTableStmt) (PlanNode, error) {
	return &CreateTablePlan{
		Table:   stmt.Table,
		Columns: stmt.Columns,
	}, nil
}

func (p *Planner) buildCreateIndexPlan(stmt *CreateIndexStmt) (PlanNode, error) {
	return &CreateIndexPlan{
		Table:  stmt.Table,
		Column: stmt.Column,
	}, nil
}

func (p *Planner) buildDropTablePlan(stmt *DropTableStmt) (PlanNode, error) {
	return &DropTablePlan{
		Table: stmt.Table,
	}, nil
}

func (p *Planner) buildDropIndexPlan(stmt *DropIndexStmt) (PlanNode, error) {
	return &DropIndexPlan{
		Table:  stmt.Table,
		Column: stmt.Column,
	}, nil
}

type TableScanPlan struct {
	Table   string
	Columns []string
	Filter  Expr
	OrderBy []OrderByCol
	Limit   int
}

func (p *TableScanPlan) String() string {
	return fmt.Sprintf("TableScan(%s)", p.Table)
}

type InsertPlan struct {
	Table string
	Key   string
	Value string
}

func (p *InsertPlan) String() string {
	return fmt.Sprintf("Insert(%s, %s, %s)", p.Table, p.Key, p.Value)
}

type UpdatePlan struct {
	Table      string
	SetClauses []SetClause
	Where      Expr
}

func (p *UpdatePlan) String() string {
	return fmt.Sprintf("Update(%s)", p.Table)
}

type DeletePlan struct {
	Table string
	Where Expr
}

func (p *DeletePlan) String() string {
	return fmt.Sprintf("Delete(%s)", p.Table)
}

type CreateTablePlan struct {
	Table   string
	Columns []string
}

func (p *CreateTablePlan) String() string {
	return fmt.Sprintf("CreateTable(%s)", p.Table)
}

type CreateIndexPlan struct {
	Table  string
	Column string
}

func (p *CreateIndexPlan) String() string {
	return fmt.Sprintf("CreateIndex(%s, %s)", p.Table, p.Column)
}

type DropTablePlan struct {
	Table string
}

func (p *DropTablePlan) String() string {
	return fmt.Sprintf("DropTable(%s)", p.Table)
}

type DropIndexPlan struct {
	Table  string
	Column string
}

func (p *DropIndexPlan) String() string {
	return fmt.Sprintf("DropIndex(%s, %s)", p.Table, p.Column)
}

type IndexScanPlan struct {
	Table    string
	IndexCol string
	EQValue  string
	Filter   Expr
	OrderBy  []OrderByCol
	Limit    int
}

func (p *IndexScanPlan) String() string {
	return fmt.Sprintf("IndexScan(%s, %s=%s)", p.Table, p.IndexCol, p.EQValue)
}
