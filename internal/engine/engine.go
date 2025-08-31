package engine

import (
	"fmt"

	"sharkDB/internal/bptree"
	"sharkDB/internal/catalog"
	"sharkDB/internal/pager2"
)

// Engine wires pager, catalog and per-table trees. It loads the tree on-demand,
// mutates it, and persists after write operations.

type Engine struct {
	p *pager2.Pager
	c *catalog.Catalog
}

func New(p *pager2.Pager) *Engine {
	return &Engine{p: p, c: catalog.New(p)}
}

func (e *Engine) Create(table string) (string, error) {
	if err := e.c.CreateTable(table); err != nil {
		return "", err
	}
	return fmt.Sprintf("Table %s created", table), nil
}

func (e *Engine) Insert(table, key, value string) (string, error) {
	id, ok := e.c.GetTableID(table)
	if !ok {
		return "", fmt.Errorf("table %s not found", table)
	}
	tree, err := e.c.LoadTree(id)
	if err != nil {
		return "", err
	}
	tree.Insert(key, value)
	if err := e.c.StoreTree(id, tree); err != nil {
		return "", err
	}
	return "OK", nil
}

func (e *Engine) Get(table, key string) (string, error) {
	id, ok := e.c.GetTableID(table)
	if !ok {
		return "", fmt.Errorf("table %s not found", table)
	}
	tree, err := e.c.LoadTree(id)
	if err != nil {
		return "", err
	}
	v, ok := tree.Get(key)
	if !ok {
		return "", bptree.ErrKeyNotFound
	}
	return v, nil
}

func (e *Engine) Update(table, key, value string) (string, error) {
	// Upsert semantics
	return e.Insert(table, key, value)
}

func (e *Engine) Delete(table, key string) (string, error) {
	id, ok := e.c.GetTableID(table)
	if !ok {
		return "", fmt.Errorf("table %s not found", table)
	}
	tree, err := e.c.LoadTree(id)
	if err != nil {
		return "", err
	}
	if ok := tree.Delete(key); !ok {
		return "", bptree.ErrKeyNotFound
	}
	if err := e.c.StoreTree(id, tree); err != nil {
		return "", err
	}
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

func (e *Engine) Scan(table, start string, limit int) ([][2]string, error) {
	id, ok := e.c.GetTableID(table)
	if !ok {
		return nil, fmt.Errorf("table %s not found", table)
	}
	tree, err := e.c.LoadTree(id)
	if err != nil {
		return nil, err
	}
	return tree.RangeFrom(start, limit), nil
}

func (e *Engine) Count(table string) (int, error) {
	id, ok := e.c.GetTableID(table)
	if !ok {
		return 0, fmt.Errorf("table %s not found", table)
	}
	tree, err := e.c.LoadTree(id)
	if err != nil {
		return 0, err
	}
	pairs := tree.RangeFrom("", 0)
	return len(pairs), nil
}

func (e *Engine) Exists(table, key string) (bool, error) {
	id, ok := e.c.GetTableID(table)
	if !ok {
		return false, fmt.Errorf("table %s not found", table)
	}
	tree, err := e.c.LoadTree(id)
	if err != nil {
		return false, err
	}
	_, ok = tree.Get(key)
	return ok, nil
}

func (e *Engine) PrefixScan(table, prefix string, limit int) ([][2]string, error) {
	id, ok := e.c.GetTableID(table)
	if !ok {
		return nil, fmt.Errorf("table %s not found", table)
	}
	tree, err := e.c.LoadTree(id)
	if err != nil {
		return nil, err
	}
	return tree.RangePrefix(prefix, limit), nil
}

func (e *Engine) Rename(oldName, newName string) (string, error) {
	if err := e.c.RenameTable(oldName, newName); err != nil {
		return "", err
	}
	return fmt.Sprintf("Table %s renamed to %s", oldName, newName), nil
}

func (e *Engine) Truncate(table string) (string, error) {
	id, ok := e.c.GetTableID(table)
	if !ok {
		return "", fmt.Errorf("table %s not found", table)
	}
	// Replace with a fresh empty tree
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

func (e *Engine) Stats(table string) (Stats, error) {
	var s Stats
	id, ok := e.c.GetTableID(table)
	if !ok {
		return s, fmt.Errorf("table %s not found", table)
	}
	tree, err := e.c.LoadTree(id)
	if err != nil {
		return s, err
	}
	s.Count = len(tree.RangeFrom("", 0))
	s.Height = tree.Height()
	if k, ok := tree.LeftmostKey(); ok {
		s.MinKey = k
	}
	if k, ok := tree.RightmostKey(); ok {
		s.MaxKey = k
	}
	return s, nil
}
