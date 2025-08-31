sharkDB
========

A tiny educational database you can run from a CLI or as a network server. It persists data to disk, supports a simple command set, and uses a minimal B+ tree per table under the hood.

Highlights
---------
- Minimal in-memory B+ tree (string keys/values) per table, serialized to disk
- Advanced page-based persistence with write-ahead logging (WAL) for crash recovery
- Table catalog (name → id) managed in metadata
- Basic transactions: `BEGIN`/`COMMIT`/`ABORT` with a coarse global write lock
- CLI REPL to run commands interactively
- TCP server mode for network access
- HTTP server mode with REST-style API
- Authentication support for both TCP and HTTP servers
- Read-only server modes for safe deployments
- Rich command set including scans, statistics, and data import/export

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

Server modes
-----------
```bash
# TCP server
./sharkdb -serve :8080
./sharkdb -serve :8080 -auth mytoken -readonly

# HTTP server  
./sharkdb -http :8090
./sharkdb -http :8090 -httpauth mytoken -httpreadonly

# Combined (both servers)
./sharkdb -serve :8080 -http :8090
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

# List all tables
sharkdb> TABLES
users
products

# Scan table with optional start key and limit
sharkdb> SCAN users alice 10
alice {"name":"Alice","age":26}
bob {"name":"Bob","age":30}

# Prefix scan
sharkdb> PREFIXSCAN users al 5
alice {"name":"Alice","age":26}
alex {"name":"Alex","age":28}

# Check if key exists
sharkdb> EXISTS users alice
true

# Get table statistics
sharkdb> STATS users
Table: users
Rows: 150
Height: 3
Leftmost key: alice
Rightmost key: zoe

# Count rows in table
sharkdb> COUNT users
150

# Export table to file
sharkdb> DUMP users users_backup.txt
Exported 150 rows to users_backup.txt

# Import table from file
sharkdb> BEGIN
OK
sharkdb(tx)> LOAD users users_backup.txt
Loaded 150 rows into users
sharkdb(tx)> COMMIT
OK

# Rename table
sharkdb> BEGIN
OK
sharkdb(tx)> RENAME users customers
Table users renamed to customers
sharkdb(tx)> COMMIT
OK

# Truncate table (delete all rows)
sharkdb> BEGIN
OK
sharkdb(tx)> TRUNCATE customers
Table customers truncated (0 rows)
sharkdb(tx)> COMMIT
OK
```

Command reference
-----------------
**Core operations:**
- CREATE `<table>`: create a new table
- INSERT `<table>` `<key>` `<value...>`: upsert a key/value row
- GET `<table>` `<key>`: fetch value by key
- UPDATE `<table>` `<key>` `<value...>`: upsert a key/value row
- DELETE `<table>` `<key>`: delete a row by key
- DELETE `<table>`: drop a table (shorthand for DROP)
- DROP `<table>`: drop a table

**Transaction management:**
- BEGIN `[READONLY]`: start a transaction; writes require a non-READONLY tx
- COMMIT: commit current transaction
- ABORT: abort current transaction

**Query and inspection:**
- TABLES: list all table names
- SCAN `<table>` `[start]` `[limit]`: scan table from start key (optional limit)
- PREFIXSCAN `<table>` `<prefix>` `[limit]`: scan keys with given prefix
- EXISTS `<table>` `<key>`: check if key exists in table
- COUNT `<table>`: count rows in table
- STATS `<table>`: show table statistics (rows, height, key range)

**Data management:**
- DUMP `<table>` `[file]`: export table to file (default: table name + .txt)
- LOAD `<table>` `<file>`: import table from file
- RENAME `<old>` `<new>`: rename a table
- TRUNCATE `<table>`: delete all rows from table

**Utility:**
- HELP: show command help
- EXIT/QUIT: exit the program

**Server-specific:**
- AUTH `<token>`: authenticate with server (TCP mode only)

Notes:
- Write operations (CREATE/INSERT/UPDATE/DELETE/DROP/RENAME/TRUNCATE/LOAD) must be inside `BEGIN` … `COMMIT`.
- Read operations (GET/TABLES/SCAN/PREFIXSCAN/EXISTS/COUNT/STATS) can be executed outside a transaction.
- Server modes support all commands except EXIT/QUIT.

Persistence
-----------
- **Page-based storage**: Data is stored in fixed 4KB pages with a free list for efficient allocation
- **Write-Ahead Log (WAL)**: All writes are logged to `sharkdb.gob.wal` before being persisted, ensuring crash recovery
- **Metadata**: Table catalog and allocation info stored in page 0
- **Blob chains**: Large table data is stored across multiple pages using linked chains
- **Page cache**: LRU cache for frequently accessed pages
- **Crash recovery**: WAL replay on startup ensures data consistency

Server APIs
-----------
**TCP Server** (`-serve :port`):
- Plain text protocol, one command per line
- Per-connection transaction state
- Authentication: `AUTH <token>` command
- Read-only mode: `-readonly` flag blocks all writes

**HTTP Server** (`-http :port`):
- REST-style API endpoints:
  - `GET /tables` - list all tables
  - `POST /tables?name=<table>` - create table
  - `DELETE /tables/<table>` - drop table
  - `GET /kv/<table>/<key>` - get value
  - `PUT /kv/<table>/<key>` - set value
  - `DELETE /kv/<table>/<key>` - delete value
  - `GET /scan/<table>?start=<key>&limit=<n>` - scan table
  - `GET /prefix/<table>?prefix=<p>&limit=<n>` - prefix scan
  - `GET /stats/<table>` - table statistics
- Authentication: `Authorization: Bearer <token>` header
- Read-only mode: `-httpreadonly` flag blocks all writes

Architecture overview
---------------------
- **cmd/sharkdb**: CLI REPL, server modes, and transaction flow
- **internal/parser**: parses text into commands
- **internal/engine**: executes commands; loads/mutates/stores table trees
- **internal/catalog**: table catalog; (de)serializes trees via the pager
- **internal/pager2**: advanced page-based persistence with WAL and crash recovery
- **internal/bptree**: in-memory B+ tree implementation
- **internal/txn**: coarse transaction manager (single writer lock)
- **internal/server**: TCP server implementation
- **internal/httpserver**: HTTP server implementation
- **internal/freelist**: placeholder for future page-based allocator

Roadmap
-------
- Finer-grained concurrency (page latches) and/or MVCC
- Schema support, secondary indexes, range scans
- Basic SQL subset (parser/planner/executor)
- Connection pooling and connection limits
- Metrics and monitoring endpoints
- Backup and restore utilities

Requirements
------------
- Go 1.21+
- Windows, macOS, or Linux

License
-------
MIT (add a LICENSE file if you plan to distribute)


