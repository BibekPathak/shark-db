# Contributing to sharkDB

Thank you for your interest in contributing to sharkDB! This document provides guidelines and information for contributors.

## Getting Started

### Prerequisites
- Go 1.21 or later
- Git
- Make (optional, for using the Makefile)

### Setting up the development environment

1. **Fork and clone the repository**
   ```bash
   git clone https://github.com/your-username/sharkDB.git
   cd sharkDB
   ```

2. **Build the project**
   ```bash
   make build
   # or
   go build ./cmd/sharkdb
   ```

3. **Run tests**
   ```bash
   make test
   # or
   go test ./...
   ```

## Project Structure

```
sharkDB/
├── cmd/sharkdb/          # Main application entry point
├── internal/             # Internal packages
│   ├── bptree/          # B+ tree implementation
│   ├── catalog/         # Table catalog management
│   ├── engine/          # Database engine
│   ├── httpserver/      # HTTP server implementation
│   ├── pager2/          # Page-based persistence with WAL
│   ├── parser/          # Command parsing
│   ├── server/          # TCP server implementation
│   └── txn/             # Transaction management
├── examples/            # Demo scripts and examples
├── Makefile            # Build and development commands
├── README.md           # Project documentation
└── LICENSE             # MIT License
```

## Development Guidelines

### Code Style
- Follow Go's official [Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- Use `gofmt` to format your code
- Keep functions small and focused
- Add comments for exported functions and complex logic

### Testing
- Write tests for new functionality
- Ensure all tests pass before submitting a PR
- Use descriptive test names
- Test both success and error cases

### Error Handling
- Always check and handle errors appropriately
- Provide meaningful error messages
- Use `fmt.Errorf` with context when wrapping errors

### Performance
- Consider performance implications of changes
- Use benchmarks for performance-critical code
- Profile code when optimizing

## Making Changes

### 1. Create a feature branch
```bash
git checkout -b feature/your-feature-name
```

### 2. Make your changes
- Write your code following the guidelines above
- Add tests for new functionality
- Update documentation if needed

### 3. Test your changes
```bash
make test
make build
```

### 4. Commit your changes
```bash
git add .
git commit -m "Add feature: brief description"
```

### 5. Push and create a pull request
```bash
git push origin feature/your-feature-name
```

## Areas for Contribution

### High Priority
- **Testing**: Add comprehensive test coverage
- **Documentation**: Improve code comments and user documentation
- **Error Handling**: Better error messages and recovery
- **Performance**: Optimize B+ tree operations and persistence

### Medium Priority
- **SQL Parser**: Implement basic SQL subset
- **Secondary Indexes**: Add support for secondary indexes
- **Connection Pooling**: Improve server performance
- **Monitoring**: Add metrics and health checks

### Low Priority
- **Schema Support**: Add data type validation
- **Backup/Restore**: Implement backup utilities
- **Replication**: Add basic replication support
- **CLI Improvements**: Better user experience

## Bug Reports

When reporting bugs, please include:
- Operating system and Go version
- Steps to reproduce the issue
- Expected vs actual behavior
- Any error messages or logs
- Sample data if relevant

## Feature Requests

When requesting features, please:
- Describe the use case clearly
- Explain why this feature would be valuable
- Consider implementation complexity
- Suggest potential approaches if you have ideas

## Code Review Process

1. **Submit a pull request** with a clear description
2. **Ensure CI passes** (tests, formatting, etc.)
3. **Address review comments** promptly
4. **Squash commits** if requested
5. **Wait for approval** before merging

## Getting Help

- **Issues**: Use GitHub issues for bugs and feature requests
- **Discussions**: Use GitHub discussions for questions and ideas
- **Code**: Review existing code for examples and patterns

## License

By contributing to sharkDB, you agree that your contributions will be licensed under the MIT License.

Thank you for contributing to sharkDB! 🐋
