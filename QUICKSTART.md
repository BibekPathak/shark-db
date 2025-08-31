# Quick Start Guide

Get sharkDB up and running in minutes!

## Prerequisites

- **Go 1.21+** installed on your system
- **Git** for cloning the repository
- **Make** (optional, for using the Makefile)

## Installation

### Option 1: Clone and Build
```bash
# Clone the repository
git clone https://github.com/your-username/sharkDB.git
cd sharkDB

# Build the binary
make build
# or
go build ./cmd/sharkdb
```

### Option 2: Download Pre-built Binary
Check the [Releases](https://github.com/your-username/sharkDB/releases) page for pre-built binaries for your platform.

## Your First sharkDB Session

### 1. Start sharkDB CLI
```bash
./sharkdb
```

You'll see:
```
sharkDB ready. Commands: CREATE/INSERT/GET/UPDATE/DELETE/BEGIN/COMMIT/ABORT. Ctrl+C to exit.
sharkdb>
```

### 2. Create a Table and Add Data
```bash
sharkdb> BEGIN
OK
sharkdb(tx)> CREATE users
Table users created
sharkdb(tx)> INSERT users alice {"name":"Alice","age":25,"email":"alice@example.com"}
OK
sharkdb(tx)> INSERT users bob {"name":"Bob","age":30,"email":"bob@example.com"}
OK
sharkdb(tx)> COMMIT
OK
```

### 3. Query Your Data
```bash
sharkdb> GET users alice
{"name":"Alice","age":25,"email":"alice@example.com"}

sharkdb> TABLES
users

sharkdb> SCAN users
alice {"name":"Alice","age":25,"email":"alice@example.com"}
bob {"name":"Bob","age":30,"email":"bob@example.com"}
```

## Server Modes

### TCP Server
```bash
# Start TCP server
./sharkdb -serve :8080

# Connect with netcat
echo "TABLES" | nc localhost 8080
```

### HTTP Server
```bash
# Start HTTP server
./sharkdb -http :8090

# Use curl to interact
curl http://localhost:8090/tables
curl -X POST "http://localhost:8090/tables?name=products"
curl -X PUT http://localhost:8090/kv/users/alice -d '{"name":"Alice","age":26}'
```

### With Authentication
```bash
# TCP server with auth
./sharkdb -serve :8080 -auth mysecret123

# HTTP server with auth
./sharkdb -http :8090 -httpauth mysecret123

# Connect with auth
echo "AUTH mysecret123" | nc localhost 8080
curl -H "Authorization: Bearer mysecret123" http://localhost:8090/tables
```

## Common Use Cases

### 1. Simple Key-Value Store
```bash
sharkdb> BEGIN
OK
sharkdb(tx)> CREATE config
Table config created
sharkdb(tx)> INSERT config app_name "MyApp"
OK
sharkdb(tx)> INSERT config version "1.0.0"
OK
sharkdb(tx)> INSERT config debug "true"
OK
sharkdb(tx)> COMMIT
OK

sharkdb> GET config app_name
"MyApp"
```

### 2. User Management
```bash
sharkdb> BEGIN
OK
sharkdb(tx)> CREATE users
Table users created
sharkdb(tx)> INSERT users user1 {"id":"user1","name":"John Doe","role":"admin","created":"2024-01-01"}
OK
sharkdb(tx)> INSERT users user2 {"id":"user2","name":"Jane Smith","role":"user","created":"2024-01-02"}
OK
sharkdb(tx)> COMMIT
OK

sharkdb> PREFIXSCAN users user
user1 {"id":"user1","name":"John Doe","role":"admin","created":"2024-01-01"}
user2 {"id":"user2","name":"Jane Smith","role":"user","created":"2024-01-02"}
```

### 3. Session Storage
```bash
sharkdb> BEGIN
OK
sharkdb(tx)> CREATE sessions
Table sessions created
sharkdb(tx)> INSERT sessions abc123 {"user_id":"user1","expires":"2024-12-31","ip":"192.168.1.1"}
OK
sharkdb(tx)> COMMIT
OK

sharkdb> EXISTS sessions abc123
true
sharkdb> GET sessions abc123
{"user_id":"user1","expires":"2024-12-31","ip":"192.168.1.1"}
```

## Data Import/Export

### Export Data
```bash
sharkdb> DUMP users users_backup.txt
Exported 2 rows to users_backup.txt
```

### Import Data
```bash
sharkdb> BEGIN
OK
sharkdb(tx)> CREATE users_restored
Table users_restored created
sharkdb(tx)> LOAD users_restored users_backup.txt
Loaded 2 rows into users_restored
sharkdb(tx)> COMMIT
OK
```

## Performance Tips

1. **Use Transactions**: Group related operations in transactions
2. **Batch Operations**: Use LOAD for bulk data import
3. **Server Mode**: Use server mode for multiple clients
4. **Read-Only Mode**: Use read-only mode for analytics workloads

## Troubleshooting

### Common Issues

**"database file not found"**
- sharkDB creates the database file automatically on first use
- Check file permissions in the current directory

**"connection refused"**
- Make sure the server is running on the correct port
- Check if the port is already in use

**"unauthorized"**
- Check if authentication is required
- Verify your token is correct

### Getting Help

- Check the [README.md](README.md) for detailed documentation
- Run `HELP` in the CLI for command reference
- Use the demo scripts in the `examples/` directory
- Open an issue on GitHub for bugs or feature requests

## Next Steps

- Explore the [full documentation](README.md)
- Try the [demo scripts](examples/)
- Check out the [contributing guide](CONTRIBUTING.md)
- Star the repository if you find it useful! ⭐

Happy coding with sharkDB! 🐋
