package parser

import (
    "errors"
    "fmt"
    "strings"
)

type Command struct {
    Name   string
    Args   []string
}

var (
    ErrParse = errors.New("parse error")
)

// Parse a very small command language:
// CREATE <table>
// INSERT <table> <key> <value>
// GET <table> <key>
// UPDATE <table> <key> <value>
// DELETE <table> <key>
// BEGIN [READONLY]
// COMMIT
// ABORT
func Parse(line string) (Command, error) {
    line = strings.TrimSpace(line)
    if line == "" { return Command{}, ErrParse }
    fields := splitFields(line)
    if len(fields) == 0 { return Command{}, ErrParse }
    cmd := strings.ToUpper(fields[0])
    args := fields[1:]
    switch cmd {
    case "CREATE":
        if len(args) != 1 { return Command{}, fmt.Errorf("CREATE requires 1 arg") }
    case "INSERT":
        if len(args) < 3 { return Command{}, fmt.Errorf("INSERT requires 3 args") }
        // allow spaces in value by joining tail
        args = []string{args[0], args[1], strings.Join(args[2:], " ")}
    case "GET":
        if len(args) != 2 { return Command{}, fmt.Errorf("GET requires 2 args") }
    case "UPDATE":
        if len(args) < 3 { return Command{}, fmt.Errorf("UPDATE requires 3 args") }
        args = []string{args[0], args[1], strings.Join(args[2:], " ")}
    case "DELETE":
        // Allow either DELETE <table> <key> (row delete) or DELETE <table> (drop table shorthand)
        if len(args) != 2 && len(args) != 1 { return Command{}, fmt.Errorf("DELETE requires 1 or 2 args") }
    case "DROP":
        if len(args) != 1 { return Command{}, fmt.Errorf("DROP requires 1 arg") }
    case "BEGIN":
        if len(args) > 1 { return Command{}, fmt.Errorf("BEGIN takes optional READONLY") }
        if len(args) == 1 { args[0] = strings.ToUpper(args[0]) }
    case "COMMIT", "ABORT":
        if len(args) != 0 { return Command{}, fmt.Errorf("%s takes no args", cmd) }
    default:
        return Command{}, fmt.Errorf("unknown command: %s", cmd)
    }
    return Command{Name: cmd, Args: args}, nil
}

func splitFields(s string) []string {
    // simple whitespace split respecting double quotes for the value is overkill here
    // We'll just split by spaces and re-join for value in Parse above.
    parts := strings.Fields(s)
    return parts
}


