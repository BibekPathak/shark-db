package bptree

import (
	"errors"
	"sort"
)

const Order = 4

type Version struct {
	TxID      uint64
	StartTime int64
	EndTime   int64
	Value     string
	Deleted   bool
}

type VersionedValue struct {
	Versions []Version
}

type Node struct {
	IsLeaf   bool
	Keys     []string
	Children []*Node
	Values   []VersionedValue
	Next     *Node
}

type BPTree struct {
	Root *Node
}

func New() *BPTree {
	return &BPTree{Root: &Node{IsLeaf: true}}
}

func (vv *VersionedValue) LatestValue() (string, bool) {
	if len(vv.Versions) == 0 {
		return "", false
	}
	return vv.Versions[0].Value, !vv.Versions[0].Deleted
}

func (vv *VersionedValue) VisibleAt(startTime int64) (string, bool) {
	for i := range vv.Versions {
		v := &vv.Versions[i]
		if v.StartTime <= startTime && (v.EndTime == 0 || v.EndTime > startTime) {
			if v.Deleted {
				return "", false
			}
			return v.Value, true
		}
	}
	return "", false
}

func (vv *VersionedValue) AddVersion(txID uint64, startTime int64, value string, deleted bool) {
	vv.Versions = append([]Version{{
		TxID:      txID,
		StartTime: startTime,
		EndTime:   0,
		Value:     value,
		Deleted:   deleted,
	}}, vv.Versions...)
}

func (vv *VersionedValue) MarkVersionDone(txID uint64, endTime int64) {
	for i := range vv.Versions {
		if vv.Versions[i].TxID == txID {
			vv.Versions[i].EndTime = endTime
			return
		}
	}
}

func (t *BPTree) Get(key string) (string, bool) {
	return t.GetWithTimestamp(key, 0)
}

func (t *BPTree) GetWithTimestamp(key string, startTime int64) (string, bool) {
	if t.Root == nil {
		return "", false
	}
	n := t.Root
	for !n.IsLeaf {
		idx := upperBound(n.Keys, key)
		n = n.Children[idx]
	}
	i := sort.SearchStrings(n.Keys, key)
	if i < len(n.Keys) && n.Keys[i] == key {
		return n.Values[i].VisibleAt(startTime)
	}
	return "", false
}

func (t *BPTree) Insert(key, value string) {
	t.InsertWithVersion(key, value, 0, 0)
}

func (t *BPTree) InsertWithVersion(key, value string, txID uint64, startTime int64) {
	if t.Root == nil {
		t.Root = &Node{IsLeaf: true}
	}
	root := t.Root
	if len(root.Keys) >= Order-1 && root.IsLeaf {
		left, sep, right := splitLeaf(root)
		t.Root = &Node{IsLeaf: false, Keys: []string{sep}, Children: []*Node{left, right}}
	}
	newChild, sep, grew := insertRecursive(t.Root, key, value, txID, startTime, true)
	if grew {
		t.Root = &Node{IsLeaf: false, Keys: []string{sep}, Children: []*Node{t.Root, newChild}}
	}
}

func (t *BPTree) Delete(key string) bool {
	return t.DeleteWithVersion(key, 0, 0)
}

func (t *BPTree) DeleteWithVersion(key string, txID uint64, startTime int64) bool {
	if t.Root == nil {
		return false
	}
	n := t.Root
	for !n.IsLeaf {
		idx := upperBound(n.Keys, key)
		n = n.Children[idx]
	}
	i := sort.SearchStrings(n.Keys, key)
	if i < len(n.Keys) && n.Keys[i] == key {
		n.Values[i].AddVersion(txID, startTime, "", true)
		return true
	}
	return false
}

func (t *BPTree) RangeFrom(start string, limit int) [][2]string {
	return t.RangeFromWithTimestamp(start, limit, 0)
}

func (t *BPTree) RangeFromWithTimestamp(start string, limit int, startTime int64) [][2]string {
	var results [][2]string
	if t.Root == nil {
		return results
	}
	n := t.Root
	if start == "" {
		for !n.IsLeaf {
			n = n.Children[0]
		}
	} else {
		for !n.IsLeaf {
			idx := upperBound(n.Keys, start)
			n = n.Children[idx]
		}
	}
	i := 0
	if start != "" {
		i = sort.SearchStrings(n.Keys, start)
	}
	for n != nil {
		for i < len(n.Keys) {
			if val, ok := n.Values[i].VisibleAt(startTime); ok {
				kv := [2]string{n.Keys[i], val}
				results = append(results, kv)
				if limit > 0 && len(results) >= limit {
					return results
				}
			}
			i++
		}
		n = n.Next
		i = 0
	}
	return results
}

func (t *BPTree) RangePrefix(prefix string, limit int) [][2]string {
	results := t.RangeFrom(prefix, 0)
	out := make([][2]string, 0, len(results))
	for _, kv := range results {
		if !hasPrefix(kv[0], prefix) {
			break
		}
		out = append(out, kv)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out
}

func hasPrefix(s, prefix string) bool {
	if len(prefix) == 0 {
		return true
	}
	if len(s) < len(prefix) {
		return false
	}
	return s[:len(prefix)] == prefix
}

func (t *BPTree) LeftmostKey() (string, bool) {
	if t.Root == nil {
		return "", false
	}
	n := t.Root
	for !n.IsLeaf {
		n = n.Children[0]
	}
	if len(n.Keys) == 0 {
		return "", false
	}
	return n.Keys[0], true
}

func (t *BPTree) RightmostKey() (string, bool) {
	if t.Root == nil {
		return "", false
	}
	n := t.Root
	for !n.IsLeaf {
		n = n.Children[len(n.Children)-1]
	}
	if len(n.Keys) == 0 {
		return "", false
	}
	return n.Keys[len(n.Keys)-1], true
}

func (t *BPTree) Height() int {
	if t.Root == nil {
		return 0
	}
	h := 0
	n := t.Root
	for {
		h++
		if n.IsLeaf {
			break
		}
		n = n.Children[0]
	}
	return h
}

func insertRecursive(n *Node, key, value string, txID uint64, startTime int64, isInsert bool) (*Node, string, bool) {
	if n.IsLeaf {
		i := sort.SearchStrings(n.Keys, key)
		if i < len(n.Keys) && n.Keys[i] == key {
			if isInsert {
				n.Values[i].AddVersion(txID, startTime, value, false)
			} else {
				n.Values[i].AddVersion(txID, startTime, "", true)
			}
			return nil, "", false
		}
		n.Keys = insertString(n.Keys, i, key)
		vv := VersionedValue{}
		if isInsert {
			vv.Versions = []Version{{TxID: txID, StartTime: startTime, Value: value, Deleted: false}}
		} else {
			vv.Versions = []Version{{TxID: txID, StartTime: startTime, Value: "", Deleted: true}}
		}
		n.Values = insertVV(n.Values, i, vv)
		if len(n.Keys) <= Order-1 {
			return nil, "", false
		}
		right, sep := splitLeafReturnRight(n)
		return right, sep, true
	}

	idx := upperBound(n.Keys, key)
	child := n.Children[idx]
	newChild, sep, grew := insertRecursive(child, key, value, txID, startTime, isInsert)
	if !grew {
		return nil, "", false
	}
	n.Keys = insertString(n.Keys, idx, sep)
	n.Children = insertNode(n.Children, idx+1, newChild)
	if len(n.Keys) <= Order-1 {
		return nil, "", false
	}
	right, sep2 := splitInternalReturnRight(n)
	return right, sep2, true
}

func splitLeafReturnRight(n *Node) (*Node, string) {
	_, sep, right := splitLeaf(n)
	right.Next = n.Next
	n.Next = right
	return right, sep
}

func splitLeaf(n *Node) (*Node, string, *Node) {
	mid := len(n.Keys) / 2
	right := &Node{IsLeaf: true}
	right.Keys = append(right.Keys, n.Keys[mid:]...)
	right.Values = append(right.Values, n.Values[mid:]...)
	sep := right.Keys[0]
	n.Keys = n.Keys[:mid]
	n.Values = n.Values[:mid]
	return n, sep, right
}

func splitInternalReturnRight(n *Node) (*Node, string) {
	mid := len(n.Keys) / 2
	sep := n.Keys[mid]

	right := &Node{IsLeaf: false}
	right.Keys = append(right.Keys, n.Keys[mid+1:]...)
	right.Children = append(right.Children, n.Children[mid+1:]...)

	n.Keys = n.Keys[:mid]
	n.Children = n.Children[:mid+1]
	return right, sep
}

func insertString(slice []string, idx int, val string) []string {
	slice = append(slice, "")
	copy(slice[idx+1:], slice[idx:])
	slice[idx] = val
	return slice
}

func insertVV(slice []VersionedValue, idx int, val VersionedValue) []VersionedValue {
	slice = append(slice, VersionedValue{})
	copy(slice[idx+1:], slice[idx:])
	slice[idx] = val
	return slice
}

func insertNode(slice []*Node, idx int, val *Node) []*Node {
	slice = append(slice, nil)
	copy(slice[idx+1:], slice[idx:])
	slice[idx] = val
	return slice
}

func upperBound(keys []string, key string) int {
	return sort.Search(len(keys), func(i int) bool { return keys[i] > key })
}

func (t *BPTree) Clone() *BPTree {
	if t == nil || t.Root == nil {
		return New()
	}
	visited := make(map[*Node]*Node)
	return &BPTree{Root: cloneNode(t.Root, visited)}
}

func cloneNode(n *Node, visited map[*Node]*Node) *Node {
	if n == nil {
		return nil
	}
	if v, ok := visited[n]; ok {
		return v
	}
	c := &Node{IsLeaf: n.IsLeaf}
	visited[n] = c
	c.Keys = append(c.Keys, n.Keys...)
	if n.IsLeaf {
		c.Values = make([]VersionedValue, len(n.Values))
		copy(c.Values, n.Values)
	} else {
		for _, ch := range n.Children {
			c.Children = append(c.Children, cloneNode(ch, visited))
		}
	}
	return c
}

func (t *BPTree) Count() int {
	if t.Root == nil {
		return 0
	}
	count := 0
	n := t.Root
	for !n.IsLeaf {
		n = n.Children[0]
	}
	for n != nil {
		for i := range n.Keys {
			if _, ok := n.Values[i].LatestValue(); ok {
				count++
			}
		}
		n = n.Next
	}
	return count
}

var ErrKeyNotFound = errors.New("key not found")
