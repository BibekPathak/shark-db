package txn

import (
    "sync"
)

// Minimal transaction manager providing per-DB global write lock and
// optimistic read concurrency. It exposes a Tx struct for grouping ops.

type Manager struct {
    writeMu sync.Mutex
}

type Tx struct {
    m *Manager
    // RW-only vs write tx distinction is implicit by whether the caller takes the lock.
    writeHeld bool
}

func NewManager() *Manager { return &Manager{} }

func (m *Manager) Begin(readOnly bool) *Tx {
    tx := &Tx{m: m}
    if !readOnly {
        m.writeMu.Lock()
        tx.writeHeld = true
    }
    return tx
}

func (t *Tx) Commit() {
    if t.writeHeld {
        t.m.writeMu.Unlock()
        t.writeHeld = false
    }
}

func (t *Tx) Abort() {
    if t.writeHeld {
        t.m.writeMu.Unlock()
        t.writeHeld = false
    }
}


