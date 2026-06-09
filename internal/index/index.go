package index

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"sharkDB/internal/bptree"
	"sharkDB/internal/pager2"
)

type Index struct {
	ID      uint64
	TableID uint64
	Column  string
	Unique  bool
}

type Manager struct {
	pager   *pager2.Pager
	indexes map[uint64][]*Index
	nextIdx uint64
}

func NewManager(p *pager2.Pager) *Manager {
	return &Manager{
		pager:   p,
		indexes: make(map[uint64][]*Index),
	}
}

func (m *Manager) indexPath(tableID uint64, column string) string {
	return fmt.Sprintf("idx_%d_%s.gob", tableID, column)
}

func (m *Manager) CreateIndex(tableID uint64, column string, unique bool) (*Index, error) {
	idx := &Index{
		ID:      m.nextIdx,
		TableID: tableID,
		Column:  column,
		Unique:  unique,
	}
	m.nextIdx++

	if m.indexes[tableID] == nil {
		m.indexes[tableID] = []*Index{}
	}
	m.indexes[tableID] = append(m.indexes[tableID], idx)

	return idx, nil
}

func (m *Manager) GetIndexes(tableID uint64) []*Index {
	return m.indexes[tableID]
}

func (m *Manager) GetIndexByColumn(tableID uint64, column string) *Index {
	for _, idx := range m.indexes[tableID] {
		if idx.Column == column {
			return idx
		}
	}
	return nil
}

type rowFetcher interface {
	GetRows(tableID uint64) ([][2]string, error)
}

func (m *Manager) BuildIndex(idx *Index, engine interface{}) error {
	ef, ok := engine.(rowFetcher)
	if !ok {
		return fmt.Errorf("engine does not support GetRows")
	}

	rows, err := ef.GetRows(idx.TableID)
	if err != nil {
		return err
	}

	tree := bptree.New()
	for _, row := range rows {
		key := row[0]
		value := row[1]

		indexedVal := extractField(value, idx.Column)
		if indexedVal == "" {
			continue
		}

		if idx.Unique {
			tree.Insert(indexedVal, key)
		} else {
			if existing, ok := tree.Get(indexedVal); ok {
				pks := strings.Split(existing, ",")
				pks = append(pks, key)
				tree.Insert(indexedVal, strings.Join(pks, ","))
			} else {
				tree.Insert(indexedVal, key)
			}
		}
	}

	return m.storeIndexTree(idx, tree)
}

func (m *Manager) storeIndexTree(idx *Index, tree *bptree.BPTree) error {
	path := m.indexPath(idx.TableID, idx.Column)

	data, err := json.Marshal(tree)
	if err != nil {
		return err
	}

	p, err := pager2.Open(path)
	if err != nil {
		return err
	}

	err = p.StoreTableBlob(idx.ID, data)
	return err
}

func (m *Manager) loadIndexTree(idx *Index) (*bptree.BPTree, error) {
	path := m.indexPath(idx.TableID, idx.Column)

	p, err := pager2.Open(path)
	if err != nil {
		return nil, err
	}

	blob, ok := p.LoadTableBlob(idx.ID)
	if !ok || len(blob) == 0 {
		return bptree.New(), nil
	}

	var tree bptree.BPTree
	if err := json.Unmarshal(blob, &tree); err != nil {
		return nil, err
	}

	return &tree, nil
}

func (m *Manager) IndexInsert(idx *Index, indexedValue string, pk string) error {
	tree, err := m.loadIndexTree(idx)
	if err != nil {
		return err
	}

	if idx.Unique {
		tree.Insert(indexedValue, pk)
	} else {
		if existing, ok := tree.Get(indexedValue); ok {
			pks := strings.Split(existing, ",")
			for _, p := range pks {
				if p == pk {
					return nil
				}
			}
			pks = append(pks, pk)
			tree.Insert(indexedValue, strings.Join(pks, ","))
		} else {
			tree.Insert(indexedValue, pk)
		}
	}

	return m.storeIndexTree(idx, tree)
}

func (m *Manager) IndexDelete(idx *Index, indexedValue string, pk string) error {
	tree, err := m.loadIndexTree(idx)
	if err != nil {
		return err
	}

	if idx.Unique {
		tree.Delete(indexedValue)
	} else {
		if existing, ok := tree.Get(indexedValue); ok {
			pks := strings.Split(existing, ",")
			newPKs := make([]string, 0)
			for _, p := range pks {
				if p != pk {
					newPKs = append(newPKs, p)
				}
			}
			if len(newPKs) == 0 {
				tree.Delete(indexedValue)
			} else {
				tree.Insert(indexedValue, strings.Join(newPKs, ","))
			}
		}
	}

	return m.storeIndexTree(idx, tree)
}

func (m *Manager) FindByIndexedValue(idx *Index, indexedValue string) ([]string, error) {
	tree, err := m.loadIndexTree(idx)
	if err != nil {
		return nil, err
	}

	if idx.Unique {
		pk, ok := tree.Get(indexedValue)
		if ok {
			return []string{pk}, nil
		}
		return nil, nil
	}

	pksStr, ok := tree.Get(indexedValue)
	if !ok {
		return nil, nil
	}
	return strings.Split(pksStr, ","), nil
}

func (m *Manager) DropIndex(idx *Index) error {
	path := m.indexPath(idx.TableID, idx.Column)
	os.Remove(path)

	tableIndexes := m.indexes[idx.TableID]
	newIndexes := make([]*Index, 0)
	for _, i := range tableIndexes {
		if i.Column != idx.Column {
			newIndexes = append(newIndexes, i)
		}
	}
	m.indexes[idx.TableID] = newIndexes

	return nil
}

func extractField(jsonStr string, field string) string {
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
