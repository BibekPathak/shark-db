package engine

import (
	"fmt"

	"sharkDB/internal/bptree"
	"sharkDB/internal/catalog"
	"sharkDB/internal/pager"
)

// Engine wires pager, catalog and per-table trees. It loads the tree on-demand,
// mutates it, and persists after write operations.

type Engine struct {
	p *pager.Pager
	c *catalog.Catalog
}

func New(p *pager.Pager) *Engine {
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
