package bptree

import (
    "errors"
    "sort"
)

// A simple in-memory B+ tree for string keys and string values.
// This is a minimal educational implementation sufficient for basic
// CREATE/INSERT/GET/UPDATE/DELETE operations used by sharkDB. Persistence
// is handled at a higher layer by serializing the tree state.

// Order defines the maximum number of keys per node before a split.
const Order = 4

type Node struct {
    IsLeaf   bool
    Keys     []string
    Children []*Node   // for internal nodes: child pointers of length len(Keys)+1
    Values   []string  // for leaf nodes: values aligned with Keys
    Next     *Node     // leaf-level linked list (for range scans)
}

type BPTree struct {
    Root *Node
}

func New() *BPTree {
    return &BPTree{Root: &Node{IsLeaf: true}}
}

// Get returns the value for key, or empty string and false if not found.
func (t *BPTree) Get(key string) (string, bool) {
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
        return n.Values[i], true
    }
    return "", false
}

// Insert sets key to value (upsert semantics).
func (t *BPTree) Insert(key, value string) {
    if t.Root == nil {
        t.Root = &Node{IsLeaf: true}
    }
    root := t.Root
    if len(root.Keys) >= Order-1 && root.IsLeaf {
        // Preemptive split of a full leaf root for simpler logic
        left, sep, right := splitLeaf(root)
        t.Root = &Node{IsLeaf: false, Keys: []string{sep}, Children: []*Node{left, right}}
    }
    newChild, sep, grew := insertRecursive(t.Root, key, value)
    if grew {
        // Root split
        t.Root = &Node{IsLeaf: false, Keys: []string{sep}, Children: []*Node{t.Root, newChild}}
    }
}

// Delete removes key if present. Returns true if deleted.
// For simplicity, this implementation supports delete by marking removal
// in leaf; it does not perform merge/redistribution underflow handling.
func (t *BPTree) Delete(key string) bool {
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
        n.Keys = append(n.Keys[:i], n.Keys[i+1:]...)
        n.Values = append(n.Values[:i], n.Values[i+1:]...)
        return true
    }
    return false
}

// insertRecursive inserts into subtree rooted at n. If the child grew and split,
// returns (newRightChild, separatorKey, grew=true). For leaves, grew indicates a split occurred.
func insertRecursive(n *Node, key, value string) (*Node, string, bool) {
    if n.IsLeaf {
        i := sort.SearchStrings(n.Keys, key)
        if i < len(n.Keys) && n.Keys[i] == key {
            n.Values[i] = value
            return nil, "", false
        }
        n.Keys = insertString(n.Keys, i, key)
        n.Values = insertString(n.Values, i, value)
        if len(n.Keys) <= Order-1 {
            return nil, "", false
        }
        right, sep := splitLeafReturnRight(n)
        return right, sep, true
    }

    // Internal node: descend
    idx := upperBound(n.Keys, key)
    child := n.Children[idx]
    newChild, sep, grew := insertRecursive(child, key, value)
    if !grew {
        return nil, "", false
    }
    // Insert separator and newChild after idx
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
    // Link leaves
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

func insertNode(slice []*Node, idx int, val *Node) []*Node {
    slice = append(slice, nil)
    copy(slice[idx+1:], slice[idx:])
    slice[idx] = val
    return slice
}

func upperBound(keys []string, key string) int {
    // first index with keys[i] > key
    return sort.Search(len(keys), func(i int) bool { return keys[i] > key })
}

// Clone performs a deep copy of the tree.
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
        c.Values = append(c.Values, n.Values...)
        // Do not clone Next chain to avoid cycles; it will be rebuilt on splits
    } else {
        for _, ch := range n.Children {
            c.Children = append(c.Children, cloneNode(ch, visited))
        }
    }
    return c
}

// For callers that want to enforce key presence
var ErrKeyNotFound = errors.New("key not found")


