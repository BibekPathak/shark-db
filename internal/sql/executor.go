package sql

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"sharkDB/internal/txn"
)

type QueryEngine interface {
	Scan(table, start string, limit int, tx *txn.Tx) ([][2]string, error)
	Insert(table, key, value string, tx *txn.Tx) (string, error)
	Update(table, key, value string, tx *txn.Tx) (string, error)
	Delete(table, key string, tx *txn.Tx) (string, error)
	Create(table string) (string, error)
	Drop(table string) (string, error)
	CreateIndex(table, column string) (string, error)
	DropIndex(table, column string) (string, error)
	TxBegin() *txn.Tx
}

type Executor struct {
	engine QueryEngine
	txm    *txn.Manager
}

func NewExecutor(eng QueryEngine, txm *txn.Manager) *Executor {
	return &Executor{engine: eng, txm: txm}
}

func (e *Executor) Execute(sqlStmt string) ([][]string, error) {
	parser := NewParser(sqlStmt)
	stmt, err := parser.Parse()
	if err != nil {
		return nil, fmt.Errorf("parse error: %v", err)
	}

	planner := NewPlanner(e.engine)
	plan, err := planner.BuildPlan(stmt)
	if err != nil {
		return nil, fmt.Errorf("plan error: %v", err)
	}

	return plan.Execute(e.engine)
}

func (p *TableScanPlan) Execute(eng interface{}) ([][]string, error) {
	e := eng.(QueryEngine)
	tx := e.TxBegin()
	defer tx.Abort()

	rows, err := e.Scan(p.Table, "", 0, tx)
	if err != nil {
		return nil, err
	}

	parsedRows := make([]map[string]interface{}, 0, len(rows))
	for _, row := range rows {
		var m map[string]interface{}
		json.Unmarshal([]byte(row[1]), &m)
		m["_key"] = row[0]
		parsedRows = append(parsedRows, m)
	}

	if p.Filter != nil {
		filtered := make([]map[string]interface{}, 0)
		for _, row := range parsedRows {
			if ok, _ := evaluateBool(p.Filter, row); ok {
				filtered = append(filtered, row)
			}
		}
		parsedRows = filtered
	}

	if len(p.OrderBy) > 0 {
		sort.Slice(parsedRows, func(i, j int) bool {
			return compareRows(parsedRows[i], parsedRows[j], p.OrderBy)
		})
	}

	if p.Limit > 0 && len(parsedRows) > p.Limit {
		parsedRows = parsedRows[:p.Limit]
	}

	var results [][]string
	for _, row := range parsedRows {
		if p.Columns[0] == "*" {
			for k, v := range row {
				results = append(results, []string{k, fmt.Sprintf("%v", v)})
			}
		} else {
			for _, col := range p.Columns {
				if v, ok := row[col]; ok {
					results = append(results, []string{col, fmt.Sprintf("%v", v)})
				}
			}
		}
	}

	return results, nil
}

func (p *InsertPlan) Execute(eng interface{}) ([][]string, error) {
	e := eng.(QueryEngine)
	tx := e.TxBegin()
	defer tx.Commit()

	_, err := e.Insert(p.Table, p.Key, p.Value, tx)
	if err != nil {
		return nil, err
	}

	return [][]string{{"OK"}}, nil
}

func (p *UpdatePlan) Execute(eng interface{}) ([][]string, error) {
	e := eng.(QueryEngine)
	tx := e.TxBegin()
	defer tx.Commit()

	rows, err := e.Scan(p.Table, "", 0, tx)
	if err != nil {
		return nil, err
	}

	count := 0
	for _, row := range rows {
		var m map[string]interface{}
		json.Unmarshal([]byte(row[1]), &m)
		m["_key"] = row[0]

		if p.Where != nil {
			if ok, _ := evaluateBool(p.Where, m); !ok {
				continue
			}
		}

		for _, set := range p.SetClauses {
			val, _ := evaluateExpr(set.Value, m)
			m[set.Column] = val
		}

		newValue, _ := json.Marshal(m)
		_, err := e.Update(p.Table, row[0], string(newValue), tx)
		if err != nil {
			return nil, err
		}
		count++
	}

	return [][]string{{fmt.Sprintf("Updated %d rows", count)}}, nil
}

func (p *DeletePlan) Execute(eng interface{}) ([][]string, error) {
	e := eng.(QueryEngine)
	tx := e.TxBegin()
	defer tx.Commit()

	rows, err := e.Scan(p.Table, "", 0, tx)
	if err != nil {
		return nil, err
	}

	count := 0
	for _, row := range rows {
		var m map[string]interface{}
		json.Unmarshal([]byte(row[1]), &m)
		m["_key"] = row[0]

		if p.Where != nil {
			if ok, _ := evaluateBool(p.Where, m); !ok {
				continue
			}
		}

		_, err := e.Delete(p.Table, row[0], tx)
		if err != nil {
			return nil, err
		}
		count++
	}

	return [][]string{{fmt.Sprintf("Deleted %d rows", count)}}, nil
}

func (p *CreateTablePlan) Execute(eng interface{}) ([][]string, error) {
	e := eng.(QueryEngine)
	_, err := e.Create(p.Table)
	if err != nil {
		return nil, err
	}

	return [][]string{{fmt.Sprintf("Table %s created", p.Table)}}, nil
}

func (p *CreateIndexPlan) Execute(eng interface{}) ([][]string, error) {
	e := eng.(QueryEngine)
	result, err := e.CreateIndex(p.Table, p.Column)
	if err != nil {
		return nil, err
	}

	return [][]string{{result}}, nil
}

func (p *DropTablePlan) Execute(eng interface{}) ([][]string, error) {
	e := eng.(QueryEngine)
	_, err := e.Drop(p.Table)
	if err != nil {
		return nil, err
	}

	return [][]string{{fmt.Sprintf("Table %s dropped", p.Table)}}, nil
}

func (p *DropIndexPlan) Execute(eng interface{}) ([][]string, error) {
	e := eng.(QueryEngine)
	result, err := e.DropIndex(p.Table, p.Column)
	if err != nil {
		return nil, err
	}

	return [][]string{{result}}, nil
}

func (e *Executor) TxBegin() *txn.Tx {
	return e.txm.Begin(true)
}

func getField(row map[string]interface{}, field string) (interface{}, error) {
	parts := strings.Split(field, ".")
	val := interface{}(row)
	for _, part := range parts {
		if m, ok := val.(map[string]interface{}); ok {
			if v, exists := m[part]; exists {
				val = v
			} else {
				return nil, fmt.Errorf("field not found: %s", field)
			}
		} else {
			return nil, fmt.Errorf("cannot access field on non-map: %s", field)
		}
	}
	return val, nil
}

func evaluateBool(expr Expr, row map[string]interface{}) (bool, error) {
	switch e := expr.(type) {
	case *ComparisonExpr:
		left, err := getField(row, e.Left.(*ColumnRef).Column)
		if err != nil {
			return false, err
		}
		right, _ := evaluateExpr(e.Right, row)
		return compareValues(left, right, e.Op)
	case *LogicalExpr:
		if e.Op == "NOT" {
			result, err := evaluateBool(e.Left.(*LogicalExpr), row)
			return !result, err
		}
		left, _ := evaluateBool(e.Left, row)
		right, _ := evaluateBool(e.Right, row)
		if e.Op == "AND" {
			return left && right, nil
		}
		return left || right, nil
	}
	return false, nil
}

func evaluateExpr(expr Expr, row map[string]interface{}) (interface{}, error) {
	switch e := expr.(type) {
	case *Literal:
		return e.Value, nil
	case *ColumnRef:
		return getField(row, e.Column)
	case *ComparisonExpr:
		left, _ := getField(row, e.Left.(*ColumnRef).Column)
		right, _ := evaluateExpr(e.Right, row)
		ok, _ := compareValues(left, right, e.Op)
		return ok, nil
	default:
		return nil, fmt.Errorf("unsupported expression type: %T", expr)
	}
}

func compareValues(a, b interface{}, op string) (bool, error) {
	switch op {
	case "=":
		return a == b, nil
	case "!=":
		return a != b, nil
	case "<":
		return compare(a, b) < 0, nil
	case ">":
		return compare(a, b) > 0, nil
	case "<=":
		return compare(a, b) <= 0, nil
	case ">=":
		return compare(a, b) >= 0, nil
	case "LIKE":
		pattern, ok := b.(string)
		if !ok {
			return false, fmt.Errorf("LIKE requires string pattern")
		}
		str, ok := a.(string)
		if !ok {
			return false, fmt.Errorf("LIKE requires string value")
		}
		re := regexp.MustCompile("^" + strings.ReplaceAll(pattern, "%", ".*") + "$")
		return re.MatchString(str), nil
	default:
		return false, fmt.Errorf("unsupported operator: %s", op)
	}
}

func compare(a, b interface{}) int {
	av, ok := a.(string)
	if !ok {
		return 0
	}
	bv, ok := b.(string)
	if !ok {
		return 0
	}
	if av < bv {
		return -1
	}
	if av > bv {
		return 1
	}
	return 0
}

func compareRows(row1, row2 map[string]interface{}, orderBy []OrderByCol) bool {
	for _, col := range orderBy {
		v1, _ := getField(row1, col.Column)
		v2, _ := getField(row2, col.Column)
		c := compare(v1, v2)
		if c < 0 {
			return !col.Desc
		}
		if c > 0 {
			return col.Desc
		}
	}
	return false
}
