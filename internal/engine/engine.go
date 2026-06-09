package engine

import (
	"encoding/json"
	"fmt"
	"strings"

	"sharkDB/internal/bptree"
	"sharkDB/internal/catalog"
	"sharkDB/internal/index"
	"sharkDB/internal/pager2"
	"sharkDB/internal/txn"
)

type Engine struct {
	p        *pager2.Pager
	c        *catalog.Catalog
	indexMgr *index.Manager
}

func New(p *pager2.Pager) *Engine {
	e := &Engine{p: p, c: catalog.New(p)}
	e.indexMgr = index.NewManager(p)
	return e
}

func (e *Engine) Create(table string) (string, error) {
	if err := e.c.CreateTable(table); err != nil {
		return "", err
	}
	return fmt.Sprintf("Table %s created", table), nil
}

func (e *Engine) Insert(table, key, value string, tx *txn.Tx) (string, error) {
	id, ok := e.c.GetTableID(table)
	if !ok {
		return "", fmt.Errorf("table %s not found", table)
	}
	tree, err := e.c.LoadTree(id)
	if err != nil {
		return "", err
	}
	tree.InsertWithVersion(key, value, tx.TxID(), tx.StartTime())
	if err := e.c.StoreTree(id, tree); err != nil {
		return "", err
	}

	e.updateIndexesOnWrite(id, key, "", value, "INSERT")

	return "OK", nil
}

func (e *Engine) Get(table, key string, tx *txn.Tx) (string, error) {
	id, ok := e.c.GetTableID(table)
	if !ok {
		return "", fmt.Errorf("table %s not found", table)
	}
	tree, err := e.c.LoadTree(id)
	if err != nil {
		return "", err
	}
	v, ok := tree.GetWithTimestamp(key, tx.StartTime())
	if !ok {
		return "", bptree.ErrKeyNotFound
	}
	return v, nil
}

func (e *Engine) Update(table, key, value string, tx *txn.Tx) (string, error) {
	id, ok := e.c.GetTableID(table)
	if !ok {
		return "", fmt.Errorf("table %s not found", table)
	}

	tree, err := e.c.LoadTree(id)
	if err != nil {
		return "", err
	}

	oldValue := ""
	if v, ok := tree.GetWithTimestamp(key, tx.StartTime()); ok {
		oldValue = v
	}

	tree.InsertWithVersion(key, value, tx.TxID(), tx.StartTime())
	if err := e.c.StoreTree(id, tree); err != nil {
		return "", err
	}

	e.updateIndexesOnWrite(id, key, oldValue, value, "UPDATE")

	return "OK", nil
}

func (e *Engine) Delete(table, key string, tx *txn.Tx) (string, error) {
	id, ok := e.c.GetTableID(table)
	if !ok {
		return "", fmt.Errorf("table %s not found", table)
	}
	tree, err := e.c.LoadTree(id)
	if err != nil {
		return "", err
	}

	oldValue := ""
	if v, ok := tree.GetWithTimestamp(key, tx.StartTime()); ok {
		oldValue = v
	}

	if ok := tree.DeleteWithVersion(key, tx.TxID(), tx.StartTime()); !ok {
		return "", bptree.ErrKeyNotFound
	}
	if err := e.c.StoreTree(id, tree); err != nil {
		return "", err
	}

	e.updateIndexesOnWrite(id, key, oldValue, "", "DELETE")

	return "OK", nil
}

func (e *Engine) Drop(table string) (string, error) {
	if err := e.c.DeleteTable(table); err != nil {
		return "", err
	}
	return fmt.Sprintf("Table %s dropped", table), nil
}

func (e *Engine) ListTables() []string {
	return e.c.ListTables()
}

func (e *Engine) Scan(table, start string, limit int, tx *txn.Tx) ([][2]string, error) {
	id, ok := e.c.GetTableID(table)
	if !ok {
		return nil, fmt.Errorf("table %s not found", table)
	}
	tree, err := e.c.LoadTree(id)
	if err != nil {
		return nil, err
	}
	return tree.RangeFromWithTimestamp(start, limit, tx.StartTime()), nil
}

func (e *Engine) Count(table string, tx *txn.Tx) (int, error) {
	id, ok := e.c.GetTableID(table)
	if !ok {
		return 0, fmt.Errorf("table %s not found", table)
	}
	tree, err := e.c.LoadTree(id)
	if err != nil {
		return 0, err
	}
	pairs := tree.RangeFromWithTimestamp("", 0, tx.StartTime())
	return len(pairs), nil
}

func (e *Engine) Exists(table, key string, tx *txn.Tx) (bool, error) {
	id, ok := e.c.GetTableID(table)
	if !ok {
		return false, fmt.Errorf("table %s not found", table)
	}
	tree, err := e.c.LoadTree(id)
	if err != nil {
		return false, err
	}
	_, ok = tree.GetWithTimestamp(key, tx.StartTime())
	return ok, nil
}

func (e *Engine) PrefixScan(table, prefix string, limit int, tx *txn.Tx) ([][2]string, error) {
	id, ok := e.c.GetTableID(table)
	if !ok {
		return nil, fmt.Errorf("table %s not found", table)
	}
	tree, err := e.c.LoadTree(id)
	if err != nil {
		return nil, err
	}
	results := tree.RangeFromWithTimestamp(prefix, 0, tx.StartTime())
	out := make([][2]string, 0, len(results))
	for _, kv := range results {
		if len(kv[0]) < len(prefix) || kv[0][:len(prefix)] != prefix {
			break
		}
		out = append(out, kv)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (e *Engine) Rename(oldName, newName string) (string, error) {
	if err := e.c.RenameTable(oldName, newName); err != nil {
		return "", err
	}
	return fmt.Sprintf("Table %s renamed to %s", oldName, newName), nil
}

func (e *Engine) Truncate(table string, tx *txn.Tx) (string, error) {
	id, ok := e.c.GetTableID(table)
	if !ok {
		return "", fmt.Errorf("table %s not found", table)
	}
	empty := bptree.New()
	if err := e.c.StoreTree(id, empty); err != nil {
		return "", err
	}
	return "OK", nil
}

type Stats struct {
	Count  int
	Height int
	MinKey string
	MaxKey string
}

func (e *Engine) Stats(table string, tx *txn.Tx) (Stats, error) {
	var s Stats
	id, ok := e.c.GetTableID(table)
	if !ok {
		return s, fmt.Errorf("table %s not found", table)
	}
	tree, err := e.c.LoadTree(id)
	if err != nil {
		return s, err
	}
	s.Count = tree.Count()
	s.Height = tree.Height()
	if k, ok := tree.LeftmostKey(); ok {
		s.MinKey = k
	}
	if k, ok := tree.RightmostKey(); ok {
		s.MaxKey = k
	}
	return s, nil
}

func (e *Engine) Begin(readOnly bool) *txn.Tx {
	return txn.NewManager().Begin(readOnly)
}

func (e *Engine) TxBegin() *txn.Tx {
	return txn.NewManager().Begin(true)
}

func (e *Engine) GetRows(tableID uint64) ([][2]string, error) {
	entries := e.c.ListTables()
	for _, entry := range entries {
		id, ok := e.c.GetTableID(entry)
		if !ok {
			continue
		}
		if id == tableID {
			tree, err := e.c.LoadTree(tableID)
			if err != nil {
				return nil, err
			}
			return tree.RangeFrom("", 0), nil
		}
	}
	return nil, fmt.Errorf("table not found")
}

func (e *Engine) CreateIndex(table, column string) (string, error) {
	tableID, ok := e.c.GetTableID(table)
	if !ok {
		return "", fmt.Errorf("table %s not found", table)
	}

	idx, err := e.indexMgr.CreateIndex(tableID, column, false)
	if err != nil {
		return "", err
	}

	if err := e.indexMgr.BuildIndex(idx, e); err != nil {
		return "", err
	}

	return fmt.Sprintf("Index created on %s(%s)", table, column), nil
}

func (e *Engine) DropIndex(table, column string) (string, error) {
	tableID, ok := e.c.GetTableID(table)
	if !ok {
		return "", fmt.Errorf("table %s not found", table)
	}

	idx := e.indexMgr.GetIndexByColumn(tableID, column)
	if idx == nil {
		return "", fmt.Errorf("index on %s(%s) not found", table, column)
	}

	if err := e.indexMgr.DropIndex(idx); err != nil {
		return "", err
	}

	return fmt.Sprintf("Index dropped on %s(%s)", table, column), nil
}

func (e *Engine) updateIndexesOnWrite(tableID uint64, key, oldValue, newValue string, op string) {
	indexes := e.indexMgr.GetIndexes(tableID)
	if len(indexes) == 0 {
		return
	}

	for _, idx := range indexes {
		oldVal := extractFieldValue(oldValue, idx.Column)
		newVal := extractFieldValue(newValue, idx.Column)

		switch op {
		case "INSERT":
			if newVal != "" {
				e.indexMgr.IndexInsert(idx, newVal, key)
			}
		case "UPDATE":
			if oldVal != newVal {
				if oldVal != "" {
					e.indexMgr.IndexDelete(idx, oldVal, key)
				}
				if newVal != "" {
					e.indexMgr.IndexInsert(idx, newVal, key)
				}
			}
		case "DELETE":
			if oldVal != "" {
				e.indexMgr.IndexDelete(idx, oldVal, key)
			}
		}
	}
}

func extractFieldValue(jsonStr string, field string) string {
	if jsonStr == "" {
		return ""
	}

	var m map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &m); err != nil {
		return ""
	}

	parts := strings.Split(field, ".")
	val := interface{}(m)
	for _, part := range parts {
		if mm, ok := val.(map[string]interface{}); ok {
			if v, exists := mm[part]; exists {
				val = v
			} else {
				return ""
			}
		} else {
			return ""
		}
	}

	switch v := val.(type) {
	case string:
		return v
	case float64:
		return fmt.Sprintf("%v", v)
	case bool:
		return fmt.Sprintf("%v", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}
