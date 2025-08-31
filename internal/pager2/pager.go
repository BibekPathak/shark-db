package pager2

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"io"
	"os"
	"sync"
)

const PageSize = 4096

// Meta is stored (gob-encoded) in page 0.
type Meta struct {
	Tables      map[string]uint64 // table name -> table id
	NextTableID uint64

	TableHead map[uint64]uint64 // table id -> head page id of blob chain (0 if none)
	FreeList  uint64            // head page id of free list (0 if empty)
}

type Pager struct {
	mu   sync.Mutex
	f    *os.File
	wal  *os.File
	meta Meta
	// simple page cache
	cache    map[uint64][]byte
	order    []uint64
	maxCache int
}

func Open(path string) (*Pager, error) {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return nil, err
	}
	wal, err := os.OpenFile(path+".wal", os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		f.Close()
		return nil, err
	}
	p := &Pager{f: f, wal: wal, cache: make(map[uint64][]byte), maxCache: 512}
	fi, err := f.Stat()
	if err != nil {
		f.Close()
		wal.Close()
		return nil, err
	}
	if fi.Size() < PageSize {
		// initialize new file with empty meta in page 0
		if err := ensureSize(f, PageSize); err != nil {
			f.Close()
			wal.Close()
			return nil, err
		}
		p.meta = Meta{Tables: make(map[string]uint64), TableHead: make(map[uint64]uint64)}
		if err := p.flushMeta(); err != nil {
			f.Close()
			wal.Close()
			return nil, err
		}
		return p, nil
	}
	if err := p.loadMeta(); err != nil {
		f.Close()
		wal.Close()
		return nil, err
	}
	if p.meta.Tables == nil {
		p.meta.Tables = make(map[string]uint64)
	}
	if p.meta.TableHead == nil {
		p.meta.TableHead = make(map[uint64]uint64)
	}
	// Replay any WAL on startup, then truncate the WAL
	if err := p.replayWAL(); err != nil {
		f.Close()
		wal.Close()
		return nil, err
	}
	if err := p.truncateWAL(); err != nil {
		f.Close()
		wal.Close()
		return nil, err
	}
	return p, nil
}

func ensureSize(f *os.File, size int64) error {
	if err := f.Truncate(size); err != nil {
		return err
	}
	return nil
}

func (p *Pager) loadMeta() error {
	buf := make([]byte, PageSize)
	if _, err := p.f.ReadAt(buf, 0); err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	dec := gob.NewDecoder(bytes.NewReader(buf))
	var m Meta
	if err := dec.Decode(&m); err != nil {
		return err
	}
	p.meta = m
	return nil
}

func (p *Pager) flushMeta() error {
	// gob encode into a PageSize buffer
	buf := make([]byte, PageSize)
	w := newBytesWriter(buf)
	enc := gob.NewEncoder(w)
	if err := enc.Encode(p.meta); err != nil {
		return err
	}
	if _, err := p.f.WriteAt(buf, 0); err != nil {
		return err
	}
	// don't cache meta page
	return p.f.Sync()
}

func (p *Pager) Meta() Meta {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.meta
}

func (p *Pager) UpdateMeta(mut func(m *Meta)) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	mut(&p.meta)
	return p.flushMeta()
}

// fault injection helper for WAL crash testing
func walFail(point string) {
	if os.Getenv("SHARKDB_WAL_FAIL") == point {
		os.Exit(2)
	}
}

// page cache helpers
func (p *Pager) cacheGet(pid uint64) ([]byte, bool) {
	if b, ok := p.cache[pid]; ok {
		// move to end
		for i, id := range p.order {
			if id == pid {
				p.order = append(p.order[:i], p.order[i+1:]...)
				break
			}
		}
		p.order = append(p.order, pid)
		cp := make([]byte, len(b))
		copy(cp, b)
		return cp, true
	}
	return nil, false
}

func (p *Pager) cachePut(pid uint64, page []byte) {
	if len(page) != PageSize {
		return
	}
	if _, ok := p.cache[pid]; !ok {
		p.order = append(p.order, pid)
		if len(p.order) > p.maxCache {
			old := p.order[0]
			p.order = p.order[1:]
			delete(p.cache, old)
		}
	}
	cp := make([]byte, PageSize)
	copy(cp, page)
	p.cache[pid] = cp
}

func (p *Pager) readPage(pid uint64) ([]byte, error) {
	if b, ok := p.cacheGet(pid); ok {
		return b, nil
	}
	buf := make([]byte, PageSize)
	if _, err := p.f.ReadAt(buf, int64(pid)*PageSize); err != nil {
		return nil, err
	}
	p.cachePut(pid, buf)
	return buf, nil
}

func (p *Pager) writePage(pid uint64, page []byte) error {
	if len(page) != PageSize {
		return errors.New("invalid page size")
	}
	if _, err := p.f.WriteAt(page, int64(pid)*PageSize); err != nil {
		return err
	}
	p.cachePut(pid, page)
	return nil
}

// StoreTableBlob writes the blob as a chain of pages, freeing any previous chain.
func (p *Pager) StoreTableBlob(tableID uint64, blob []byte) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	// WAL append
	if err := p.walAppendStore(tableID, blob); err != nil {
		return err
	}
	walFail("after_wal_store")
	// free old chain if exists
	if head, ok := p.meta.TableHead[tableID]; ok && head != 0 {
		if err := p.freeChain(head); err != nil {
			return err
		}
		p.meta.TableHead[tableID] = 0
	}
	if len(blob) == 0 {
		// nothing to store
		if err := p.flushMeta(); err != nil {
			return err
		}
		return p.walSync()
	}
	// write new chain
	const headerSize = 12 // next(8) + dataLen(4)
	capPerPage := PageSize - headerSize
	var pages []uint64
	for off := 0; off < len(blob); off += capPerPage {
		end := off + capPerPage
		if end > len(blob) {
			end = len(blob)
		}
		pid, err := p.allocPage()
		if err != nil {
			return err
		}
		pages = append(pages, pid)
		page := make([]byte, PageSize)
		binary.LittleEndian.PutUint32(page[8:12], uint32(end-off))
		copy(page[headerSize:], blob[off:end])
		if err := p.writePage(pid, page); err != nil {
			return err
		}
	}
	// link next pointers
	for i := 0; i < len(pages); i++ {
		var next uint64
		if i+1 < len(pages) {
			next = pages[i+1]
		} else {
			next = 0
		}
		pg, err := p.readPage(pages[i])
		if err != nil {
			return err
		}
		binary.LittleEndian.PutUint64(pg[:8], next)
		if err := p.writePage(pages[i], pg); err != nil {
			return err
		}
	}
	walFail("before_meta_flush")
	p.meta.TableHead[tableID] = pages[0]
	if err := p.flushMeta(); err != nil {
		return err
	}
	if err := p.f.Sync(); err != nil {
		return err
	}
	return p.walSync()
}

// LoadTableBlob reads the blob chain for tableID.
func (p *Pager) LoadTableBlob(tableID uint64) ([]byte, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	head := p.meta.TableHead[tableID]
	if head == 0 {
		return nil, false
	}
	const headerSize = 12
	var out []byte
	pid := head
	for pid != 0 {
		buf, err := p.readPage(pid)
		if err != nil {
			return nil, false
		}
		next := binary.LittleEndian.Uint64(buf[:8])
		n := int(binary.LittleEndian.Uint32(buf[8:12]))
		if headerSize+n > len(buf) {
			return nil, false
		}
		out = append(out, buf[headerSize:headerSize+n]...)
		pid = next
	}
	return out, true
}

// DeleteTableBlob frees the chain for tableID.
func (p *Pager) DeleteTableBlob(tableID uint64) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	// WAL append
	if err := p.walAppendDelete(tableID); err != nil {
		return err
	}
	walFail("after_wal_delete")
	head := p.meta.TableHead[tableID]
	if head == 0 {
		return p.walSync()
	}
	if err := p.freeChain(head); err != nil {
		return err
	}
	delete(p.meta.TableHead, tableID)
	if err := p.flushMeta(); err != nil {
		return err
	}
	return p.walSync()
}

func (p *Pager) freeChain(head uint64) error {
	pid := head
	for pid != 0 {
		buf, err := p.readPage(pid)
		if err != nil {
			return err
		}
		next := binary.LittleEndian.Uint64(buf[:8])
		// push pid onto free list
		binary.LittleEndian.PutUint64(buf[:8], p.meta.FreeList)
		if err := p.writePage(pid, buf); err != nil {
			return err
		}
		p.meta.FreeList = pid
		pid = next
	}
	return nil
}

func (p *Pager) allocPage() (uint64, error) {
	// pop from free list or extend file
	if p.meta.FreeList != 0 {
		pid := p.meta.FreeList
		buf, err := p.readPage(pid)
		if err != nil {
			return 0, err
		}
		p.meta.FreeList = binary.LittleEndian.Uint64(buf[:8])
		binary.LittleEndian.PutUint64(buf[:8], 0)
		if err := p.writePage(pid, buf); err != nil {
			return 0, err
		}
		return pid, nil
	}
	// allocate at end
	fi, err := p.f.Stat()
	if err != nil {
		return 0, err
	}
	size := fi.Size()
	pid := uint64(size / PageSize)
	if err := ensureSize(p.f, size+PageSize); err != nil {
		return 0, err
	}
	// page is zeroed; cache it
	p.cachePut(pid, make([]byte, PageSize))
	return pid, nil
}

// WAL helpers
func (p *Pager) walAppendStore(tableID uint64, blob []byte) error {
	// record: 1 | tableID | blobLen | blob
	hdr := make([]byte, 1+8+8)
	hdr[0] = 1
	binary.LittleEndian.PutUint64(hdr[1:9], tableID)
	binary.LittleEndian.PutUint64(hdr[9:17], uint64(len(blob)))
	if _, err := p.wal.Write(hdr); err != nil {
		return err
	}
	if len(blob) > 0 {
		if _, err := p.wal.Write(blob); err != nil {
			return err
		}
	}
	return nil
}

func (p *Pager) walAppendDelete(tableID uint64) error {
	hdr := make([]byte, 1+8+8)
	hdr[0] = 2
	binary.LittleEndian.PutUint64(hdr[1:9], tableID)
	// blobLen=0
	if _, err := p.wal.Write(hdr); err != nil {
		return err
	}
	return nil
}

func (p *Pager) walSync() error {
	if p.wal == nil {
		return nil
	}
	return p.wal.Sync()
}

func (p *Pager) truncateWAL() error {
	if p.wal == nil {
		return nil
	}
	return p.wal.Truncate(0)
}

func (p *Pager) replayWAL() error {
	if p.wal == nil {
		return nil
	}
	if _, err := p.wal.Seek(0, 0); err != nil {
		return err
	}
	// read all
	data, err := io.ReadAll(p.wal)
	if err != nil {
		return err
	}
	off := 0
	for off+17 <= len(data) {
		recType := data[off]
		tableID := binary.LittleEndian.Uint64(data[off+1 : off+9])
		blobLen := int(binary.LittleEndian.Uint64(data[off+9 : off+17]))
		off += 17
		switch recType {
		case 1:
			if off+blobLen > len(data) {
				return nil
			}
			blob := data[off : off+blobLen]
			off += blobLen
			// apply as store without logging
			// free old
			if head, ok := p.meta.TableHead[tableID]; ok && head != 0 {
				if err := p.freeChain(head); err != nil {
					return err
				}
				p.meta.TableHead[tableID] = 0
			}
			if len(blob) > 0 {
				const headerSize = 12
				capPerPage := PageSize - headerSize
				var pages []uint64
				for o := 0; o < len(blob); o += capPerPage {
					e := o + capPerPage
					if e > len(blob) {
						e = len(blob)
					}
					pid, err := p.allocPage()
					if err != nil {
						return err
					}
					pages = append(pages, pid)
					chunk := blob[o:e]
					page := make([]byte, PageSize)
					binary.LittleEndian.PutUint32(page[8:12], uint32(len(chunk)))
					copy(page[headerSize:], chunk)
					if _, err := p.f.WriteAt(page, int64(pid)*PageSize); err != nil {
						return err
					}
				}
				for i := 0; i < len(pages); i++ {
					var next uint64
					if i+1 < len(pages) {
						next = pages[i+1]
					}
					nb := make([]byte, 8)
					binary.LittleEndian.PutUint64(nb, next)
					if _, err := p.f.WriteAt(nb, int64(pages[i])*PageSize); err != nil {
						return err
					}
				}
				p.meta.TableHead[tableID] = pages[0]
			}
			if err := p.flushMeta(); err != nil {
				return err
			}
		case 2:
			if head := p.meta.TableHead[tableID]; head != 0 {
				if err := p.freeChain(head); err != nil {
					return err
				}
			}
			delete(p.meta.TableHead, tableID)
			if err := p.flushMeta(); err != nil {
				return err
			}
		default:
			return nil
		}
	}
	return nil
}

// minimal writer that writes to provided backing buffer
type bytesWriter struct {
	buf []byte
	off int
}

func newBytesWriter(buf []byte) *bytesWriter { return &bytesWriter{buf: buf} }

func (w *bytesWriter) Write(p []byte) (int, error) {
	if w.off+len(p) > len(w.buf) {
		// truncate to fit
		p = p[:len(w.buf)-w.off]
	}
	n := copy(w.buf[w.off:], p)
	w.off += n
	return n, nil
}
