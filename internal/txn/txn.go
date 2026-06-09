package txn

import (
	"sync"
	"sync/atomic"
	"time"
)

type Manager struct {
	writeMu     sync.Mutex
	tableLocks  map[string]*sync.RWMutex
	tableLockMu sync.Mutex
	txCounter   uint64
	counterMu   sync.Mutex
	activeTxs   map[uint64]*Tx
	activeMu    sync.RWMutex
	gcStopCh    chan struct{}
}

type Tx struct {
	ID         uint64
	m          *Manager
	readOnly   bool
	writeHeld  bool
	startTime  int64
	readSet    map[string]uint64
	writeSet   map[string]string
	tableLocks map[string]bool
}

func NewManager() *Manager {
	m := &Manager{
		tableLocks: make(map[string]*sync.RWMutex),
		activeTxs:  make(map[uint64]*Tx),
		gcStopCh:   make(chan struct{}),
	}
	go m.gcLoop()
	return m
}

func (m *Manager) NextTxID() uint64 {
	return atomic.AddUint64(&m.txCounter, 1)
}

func (m *Manager) Begin(readOnly bool) *Tx {
	m.counterMu.Lock()
	txID := m.NextTxID()
	startTime := time.Now().UnixNano()
	m.counterMu.Unlock()

	tx := &Tx{
		ID:         txID,
		m:          m,
		readOnly:   readOnly,
		writeHeld:  false,
		startTime:  startTime,
		readSet:    make(map[string]uint64),
		writeSet:   make(map[string]string),
		tableLocks: make(map[string]bool),
	}

	if !readOnly {
		m.writeMu.Lock()
		tx.writeHeld = true
	}

	m.activeMu.Lock()
	m.activeTxs[txID] = tx
	m.activeMu.Unlock()

	return tx
}

func (m *Manager) LockTable(tx *Tx, table string) {
	m.tableLockMu.Lock()
	lock, ok := m.tableLocks[table]
	if !ok {
		lock = &sync.RWMutex{}
		m.tableLocks[table] = lock
	}
	m.tableLockMu.Unlock()

	if tx.readOnly {
		lock.RLock()
	} else {
		lock.Lock()
	}
	tx.tableLocks[table] = true
}

func (m *Manager) GetTableLock(table string) *sync.RWMutex {
	m.tableLockMu.Lock()
	defer m.tableLockMu.Unlock()
	return m.tableLocks[table]
}

func (t *Tx) TxID() uint64 {
	return t.ID
}

func (t *Tx) StartTime() int64 {
	return t.startTime
}

func (t *Tx) Commit() {
	if t.writeHeld {
		t.m.writeMu.Unlock()
		t.writeHeld = false
	}

	t.m.activeMu.Lock()
	delete(t.m.activeTxs, t.ID)
	t.m.activeMu.Unlock()

	for table := range t.tableLocks {
		t.m.tableLockMu.Lock()
		lock := t.m.tableLocks[table]
		t.m.tableLockMu.Unlock()
		if t.readOnly {
			lock.RUnlock()
		} else {
			lock.Unlock()
		}
	}
}

func (t *Tx) Abort() {
	t.writeSet = nil

	if t.writeHeld {
		t.m.writeMu.Unlock()
		t.writeHeld = false
	}

	t.m.activeMu.Lock()
	delete(t.m.activeTxs, t.ID)
	t.m.activeMu.Unlock()

	for table := range t.tableLocks {
		t.m.tableLockMu.Lock()
		lock := t.m.tableLocks[table]
		t.m.tableLockMu.Unlock()
		if t.readOnly {
			lock.RUnlock()
		} else {
			lock.Unlock()
		}
	}
}

func (m *Manager) gcLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			m.purgeOldVersions()
		case <-m.gcStopCh:
			return
		}
	}
}

func (m *Manager) purgeOldVersions() {
	m.activeMu.RLock()
	minTxID := uint64(0)
	for id := range m.activeTxs {
		if minTxID == 0 || id < minTxID {
			minTxID = id
		}
	}
	m.activeMu.RUnlock()

	if minTxID == 0 {
		return
	}
}

func (m *Manager) StopGC() {
	close(m.gcStopCh)
}
