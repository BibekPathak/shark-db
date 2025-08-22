sharkDB
========

A tiny educational database you can run from a CLI. It persists data to disk, supports a simple command set, and uses a minimal B+ tree per table under the hood.

Highlights
---------
- Minimal in-memory B+ tree (string keys/values) per table, serialized to disk
- Simple persistence via a single GOB file (`sharkdb.gob`)
- Table catalog (name → id) managed in metadata
- Basic transactions: `BEGIN`/`COMMIT`/`ABORT` with a coarse global write lock
- CLI REPL to run commands interactively

Quick start
-----------
```bash
# from project root
go build ./cmd/sharkdb
./cmd/sharkdb/sharkdb            # Linux/macOS
# or on Windows
cmd\sharkdb\sharkdb.exe
# alternatively
go run ./cmd/sharkdb
```

Usage demo
----------
```text
sharkDB ready. Commands: CREATE/INSERT/GET/UPDATE/DELETE/BEGIN/COMMIT/ABORT. Ctrl+C to exit.

sharkdb> BEGIN
OK
sharkdb(tx)> CREATE users
Table users created
sharkdb(tx)> INSERT users alice {"name":"Alice","age":25}
OK
sharkdb(tx)> UPDATE users alice {"name":"Alice","age":26}
OK
sharkdb(tx)> COMMIT
OK

sharkdb> GET users alice
{"name":"Alice","age":26}

# Row delete
sharkdb> BEGIN
OK
sharkdb(tx)> DELETE users alice
OK
sharkdb(tx)> COMMIT
OK

# Drop table (either DROP or DELETE <table>)
sharkdb> BEGIN
OK
sharkdb(tx)> DROP users
Table users dropped
sharkdb(tx)> COMMIT
OK
```

Command reference
-----------------
- CREATE `<table>`: create a new table
- INSERT `<table>` `<key>` `<value...>`: upsert a key/value row
- GET `<table>` `<key>`: fetch value by key
- UPDATE `<table>` `<key>` `<value...>`: upsert a key/value row
- DELETE `<table>` `<key>`: delete a row by key
- DELETE `<table>`: drop a table (shorthand for DROP)
- DROP `<table>`: drop a table
- BEGIN `[READONLY]`: start a transaction; writes require a non-READONLY tx
- COMMIT: commit current transaction
- ABORT: abort current transaction

Notes:
- Write operations (CREATE/INSERT/UPDATE/DELETE/DROP) must be inside `BEGIN` … `COMMIT`.
- `GET` can be executed outside a transaction.

Persistence
-----------
- All state is stored in `sharkdb.gob` (GOB-encoded) in the current working directory:
  - Meta: table catalog (name → id) and id allocator
  - Per-table blobs: serialized B+ tree for that table

Architecture overview
---------------------
- cmd/sharkdb: CLI REPL and transaction flow
- internal/parser: parses text into commands
- internal/engine: executes commands; loads/mutates/stores table trees
- internal/catalog: table catalog; (de)serializes trees via the pager
- internal/pager: persistence of metadata and per-table blobs (GOB file)
- internal/bptree: in-memory B+ tree implementation
- internal/txn: coarse transaction manager (single writer lock)
- internal/freelist: placeholder for future page-based allocator

Roadmap
-------
- Replace GOB persistence with a real page manager and free list
- Add write-ahead logging (WAL) and crash recovery
- Finer-grained concurrency (page latches) and/or MVCC
- Schema support, secondary indexes, range scans
- Basic SQL subset (parser/planner/executor)

Requirements
------------
- Go 1.21+
- Windows, macOS, or Linux

License
-------
MIT (add a LICENSE file if you plan to distribute)


