.PHONY: build clean test run-cli run-tcp run-http run-both install

# Build the sharkDB binary
build:
	go build -o sharkdb ./cmd/sharkdb

# Clean build artifacts
clean:
	rm -f sharkdb
	rm -f sharkdb.exe
	rm -f *.gob
	rm -f *.gob.wal

# Run tests
test:
	go test ./...

# Run CLI mode
run-cli: build
	./sharkdb

# Run TCP server on port 8080
run-tcp: build
	./sharkdb -serve :8080

# Run HTTP server on port 8090
run-http: build
	./sharkdb -http :8090

# Run both servers
run-both: build
	./sharkdb -serve :8080 -http :8090

# Run with authentication
run-tcp-auth: build
	./sharkdb -serve :8080 -auth mysecret123

run-http-auth: build
	./sharkdb -http :8090 -httpauth mysecret123

# Run in read-only mode
run-tcp-readonly: build
	./sharkdb -serve :8080 -readonly

run-http-readonly: build
	./sharkdb -http :8090 -httpreadonly

# Install to system PATH (Linux/macOS)
install: build
	sudo cp sharkdb /usr/local/bin/

# Development helpers
dev-build:
	go build -race -o sharkdb ./cmd/sharkdb

dev-test:
	go test -race -v ./...

# Create release builds for multiple platforms
release: clean
	GOOS=linux GOARCH=amd64 go build -o sharkdb-linux-amd64 ./cmd/sharkdb
	GOOS=darwin GOARCH=amd64 go build -o sharkdb-darwin-amd64 ./cmd/sharkdb
	GOOS=windows GOARCH=amd64 go build -o sharkdb-windows-amd64.exe ./cmd/sharkdb
