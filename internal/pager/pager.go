package pager

import (
    "encoding/gob"
    "errors"
    "os"
    "sync"
)

// Minimal pager that persists a single database file as a GOB-encoded struct.
// This is a simplified stand-in for a page-based file manager with a free list.

type Meta struct {
    Tables map[string]uint64 // table name -> table id
    NextTableID uint64
}

type DBImage struct {
    Meta     Meta
    // Each table id maps to a blob payload (used by higher layers to store trees)
    Tables   map[uint64][]byte
    FreeIDs  []uint64 // placeholder to illustrate a freelist concept
}

type Pager struct {
    path string
    mu   sync.RWMutex
    img  DBImage
}

func Open(path string) (*Pager, error) {
    p := &Pager{path: path}
    if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
        p.img = DBImage{Meta: Meta{Tables: make(map[string]uint64)}, Tables: make(map[uint64][]byte)}
        if err := p.flush(); err != nil {
            return nil, err
        }
        return p, nil
    }
    f, err := os.Open(path)
    if err != nil {
        return nil, err
    }
    defer f.Close()
    dec := gob.NewDecoder(f)
    if err := dec.Decode(&p.img); err != nil {
        return nil, err
    }
    return p, nil
}

func (p *Pager) flush() error {
    f, err := os.Create(p.path)
    if err != nil {
        return err
    }
    defer f.Close()
    enc := gob.NewEncoder(f)
    return enc.Encode(&p.img)
}

func (p *Pager) Meta() Meta {
    p.mu.RLock()
    defer p.mu.RUnlock()
    return p.img.Meta
}

func (p *Pager) UpdateMeta(mut func(m *Meta)) error {
    p.mu.Lock()
    defer p.mu.Unlock()
    mut(&p.img.Meta)
    return p.flush()
}

// StoreTableBlob persists the serialized table state for a table id.
func (p *Pager) StoreTableBlob(tableID uint64, blob []byte) error {
    p.mu.Lock()
    defer p.mu.Unlock()
    if p.img.Tables == nil { p.img.Tables = make(map[uint64][]byte) }
    p.img.Tables[tableID] = blob
    return p.flush()
}

// LoadTableBlob returns the serialized table state for a table id.
func (p *Pager) LoadTableBlob(tableID uint64) ([]byte, bool) {
    p.mu.RLock()
    defer p.mu.RUnlock()
    blob, ok := p.img.Tables[tableID]
    return blob, ok
}

// DeleteTableBlob removes the serialized state for a table id and persists.
func (p *Pager) DeleteTableBlob(tableID uint64) error {
    p.mu.Lock()
    defer p.mu.Unlock()
    delete(p.img.Tables, tableID)
    return p.flush()
}


