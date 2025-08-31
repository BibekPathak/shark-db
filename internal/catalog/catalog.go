package catalog

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"

	"sharkDB/internal/bptree"
	"sharkDB/internal/pager2"
)

// Catalog maps table names to persistent table ids and stores/loads
// each table's B+ tree as a serialized blob via the pager.

type Catalog struct {
	p *pager2.Pager
}

func New(p *pager2.Pager) *Catalog { return &Catalog{p: p} }

func (c *Catalog) CreateTable(name string) error {
	m := c.p.Meta()
	if _, exists := m.Tables[name]; exists {
		return fmt.Errorf("table %s already exists", name)
	}
	return c.p.UpdateMeta(func(meta *pager2.Meta) {
		if meta.Tables == nil {
			meta.Tables = make(map[string]uint64)
		}
		meta.NextTableID++
		meta.Tables[name] = meta.NextTableID
	})
}

func (c *Catalog) GetTableID(name string) (uint64, bool) {
	m := c.p.Meta()
	id, ok := m.Tables[name]
	return id, ok
}

func (c *Catalog) LoadTree(tableID uint64) (*bptree.BPTree, error) {
	blob, ok := c.p.LoadTableBlob(tableID)
	if !ok || len(blob) == 0 {
		// New empty tree
		return bptree.New(), nil
	}
	var tree bptree.BPTree
	dec := gob.NewDecoder(bytes.NewReader(blob))
	if err := dec.Decode(&tree); err != nil {
		return nil, err
	}
	return &tree, nil
}

func (c *Catalog) StoreTree(tableID uint64, tree *bptree.BPTree) error {
	if tree == nil {
		return errors.New("nil tree")
	}
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(tree); err != nil {
		return err
	}
	return c.p.StoreTableBlob(tableID, buf.Bytes())
}

// DeleteTable removes table metadata and its blob.
func (c *Catalog) DeleteTable(name string) error {
	m := c.p.Meta()
	id, ok := m.Tables[name]
	if !ok {
		return fmt.Errorf("table %s not found", name)
	}
	if err := c.p.DeleteTableBlob(id); err != nil {
		return err
	}
	return c.p.UpdateMeta(func(meta *pager2.Meta) {
		delete(meta.Tables, name)
	})
}

// ListTables returns all table names.
func (c *Catalog) ListTables() []string {
	m := c.p.Meta()
	out := make([]string, 0, len(m.Tables))
	for name := range m.Tables {
		out = append(out, name)
	}
	return out
}

// RenameTable changes a table's name in metadata.
func (c *Catalog) RenameTable(oldName, newName string) error {
	m := c.p.Meta()
	if _, ok := m.Tables[newName]; ok {
		return fmt.Errorf("table %s already exists", newName)
	}
	id, ok := m.Tables[oldName]
	if !ok {
		return fmt.Errorf("table %s not found", oldName)
	}
	return c.p.UpdateMeta(func(meta *pager2.Meta) {
		delete(meta.Tables, oldName)
		if meta.Tables == nil {
			meta.Tables = make(map[string]uint64)
		}
		meta.Tables[newName] = id
	})
}
