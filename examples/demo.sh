#!/bin/bash

# sharkDB Demo Script
# This script demonstrates the key features of sharkDB

echo "🐋 sharkDB Demo"
echo "==============="
echo

# Build sharkDB if not already built
if [ ! -f "../sharkdb" ]; then
    echo "Building sharkDB..."
    cd .. && make build && cd examples
fi

# Start sharkDB in background
echo "Starting sharkDB server..."
../sharkdb -serve :8080 &
SHARKDB_PID=$!

# Wait for server to start
sleep 2

echo "Demo 1: Basic Operations"
echo "------------------------"
echo

# Create a table and insert data
echo "Creating 'users' table and inserting data..."
echo "CREATE users" | nc localhost 8080
echo "INSERT users alice '{\"name\":\"Alice\",\"age\":25,\"email\":\"alice@example.com\"}'" | nc localhost 8080
echo "INSERT users bob '{\"name\":\"Bob\",\"age\":30,\"email\":\"bob@example.com\"}'" | nc localhost 8080
echo "INSERT users charlie '{\"name\":\"Charlie\",\"age\":35,\"email\":\"charlie@example.com\"}'" | nc localhost 8080

echo
echo "Listing all tables:"
echo "TABLES" | nc localhost 8080

echo
echo "Getting user 'alice':"
echo "GET users alice" | nc localhost 8080

echo
echo "Demo 2: Scanning and Queries"
echo "----------------------------"
echo

echo "Scanning all users:"
echo "SCAN users" | nc localhost 8080

echo
echo "Prefix scan for users starting with 'a':"
echo "PREFIXSCAN users a" | nc localhost 8080

echo
echo "Checking if user exists:"
echo "EXISTS users alice" | nc localhost 8080
echo "EXISTS users dave" | nc localhost 8080

echo
echo "Getting table statistics:"
echo "STATS users" | nc localhost 8080

echo
echo "Demo 3: Data Management"
echo "----------------------"
echo

echo "Creating 'products' table:"
echo "CREATE products" | nc localhost 8080

echo "Inserting product data:"
echo "INSERT products laptop '{\"name\":\"MacBook Pro\",\"price\":1299,\"category\":\"electronics\"}'" | nc localhost 8080
echo "INSERT products phone '{\"name\":\"iPhone 15\",\"price\":799,\"category\":\"electronics\"}'" | nc localhost 8080
echo "INSERT products book '{\"name\":\"Database Design\",\"price\":49,\"category\":\"books\"}'" | nc localhost 8080

echo
echo "Listing all tables:"
echo "TABLES" | nc localhost 8080

echo
echo "Demo 4: Transactions"
echo "-------------------"
echo

echo "Starting a transaction and updating data:"
echo "BEGIN" | nc localhost 8080
echo "UPDATE users alice '{\"name\":\"Alice Smith\",\"age\":26,\"email\":\"alice.smith@example.com\"}'" | nc localhost 8080
echo "INSERT users dave '{\"name\":\"Dave Wilson\",\"age\":28,\"email\":\"dave@example.com\"}'" | nc localhost 8080
echo "COMMIT" | nc localhost 8080

echo
echo "Verifying changes:"
echo "GET users alice" | nc localhost 8080
echo "GET users dave" | nc localhost 8080

echo
echo "Demo 5: Data Export/Import"
echo "-------------------------"
echo

echo "Exporting users table:"
echo "DUMP users users_backup.txt" | nc localhost 8080

echo "Creating new table and importing data:"
echo "CREATE users_restored" | nc localhost 8080
echo "LOAD users_restored users_backup.txt" | nc localhost 8080

echo
echo "Comparing original and restored:"
echo "COUNT users" | nc localhost 8080
echo "COUNT users_restored" | nc localhost 8080

echo
echo "Demo 6: Table Management"
echo "----------------------"
echo

echo "Renaming table:"
echo "RENAME products inventory" | nc localhost 8080

echo "Listing tables after rename:"
echo "TABLES" | nc localhost 8080

echo
echo "Demo Complete!"
echo "=============="
echo

# Cleanup
echo "Cleaning up..."
kill $SHARKDB_PID
rm -f users_backup.txt
rm -f *.gob
rm -f *.gob.wal

echo "Demo finished. Check the output above to see sharkDB in action!"
