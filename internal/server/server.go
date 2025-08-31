package server

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"

	"sharkDB/internal/engine"
	"sharkDB/internal/parser"
	"sharkDB/internal/txn"
)

// Serve starts a plain TCP server that accepts one-line commands compatible with the CLI.
// Each connection maintains its own transaction state. Commands and outputs are text lines.
type Options struct {
	RequireToken string
	ReadOnly     bool
}

func Serve(addr string, eng *engine.Engine, tm *txn.Manager, opts Options) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	log.Printf("sharkDB server listening on %s", addr)
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("accept error: %v", err)
			continue
		}
		go handleConn(conn, eng, tm, opts)
	}
}

func handleConn(conn net.Conn, eng *engine.Engine, tm *txn.Manager, opts Options) {
	defer conn.Close()
	wr := bufio.NewWriter(conn)
	_, _ = fmt.Fprintln(wr, "sharkDB server ready. Send commands; close socket to exit.")
	_ = wr.Flush()
	in := bufio.NewScanner(conn)
	var inTx bool
	var writeTx bool
	var curTx *txn.Tx
	authed := opts.RequireToken == ""
	for in.Scan() {
		line := strings.TrimSpace(in.Text())
		if line == "" {
			continue
		}
		cmd, err := parser.Parse(line)
		if err != nil {
			fmt.Fprintln(wr, "ERR:", err)
			wr.Flush()
			continue
		}
		switch cmd.Name {
		case "AUTH":
			if len(cmd.Args) != 1 {
				fmt.Fprintln(wr, "ERR: AUTH <token>")
				wr.Flush()
				continue
			}
			if opts.RequireToken == "" {
				fmt.Fprintln(wr, "OK")
				wr.Flush()
				continue
			}
			if cmd.Args[0] == opts.RequireToken {
				authed = true
				fmt.Fprintln(wr, "OK")
			} else {
				fmt.Fprintln(wr, "ERR: unauthorized")
			}
			wr.Flush()
			continue
		case "HELP":
			fmt.Fprintln(wr, "Commands:")
			fmt.Fprintln(wr, "  BEGIN [READONLY] | COMMIT | ABORT")
			fmt.Fprintln(wr, "  CREATE <table> | DROP <table> | RENAME <old> <new> | TRUNCATE <table>")
			fmt.Fprintln(wr, "  INSERT <table> <key> <value> | UPDATE <table> <key> <value> | DELETE <table> [key]")
			fmt.Fprintln(wr, "  GET <table> <key> | EXISTS <table> <key>")
			fmt.Fprintln(wr, "  TABLES | SCAN <table> [start] [limit] | PREFIXSCAN <table> <prefix> [limit]")
			fmt.Fprintln(wr, "  COUNT <table> | STATS <table> | DUMP <table> [file] | LOAD <table> <file>")
			fmt.Fprintln(wr, "  HELP | EXIT | QUIT")
			wr.Flush()
			continue
		case "EXIT", "QUIT":
			fmt.Fprintln(wr, "Bye")
			wr.Flush()
			return
		case "BEGIN":
			if inTx {
				fmt.Fprintln(wr, "ERR: already in transaction")
				wr.Flush()
				continue
			}
			readOnly := len(cmd.Args) == 1 && cmd.Args[0] == "READONLY"
			if !readOnly {
				curTx = tm.Begin(false)
			} else {
				curTx = nil
			}
			inTx = true
			writeTx = !readOnly
			fmt.Fprintln(wr, "OK")
		case "COMMIT":
			if !inTx {
				fmt.Fprintln(wr, "ERR: not in transaction")
				wr.Flush()
				continue
			}
			if curTx != nil {
				curTx.Commit()
				curTx = nil
			}
			inTx = false
			writeTx = false
			fmt.Fprintln(wr, "OK")
		case "ABORT":
			if !inTx {
				fmt.Fprintln(wr, "ERR: not in transaction")
				wr.Flush()
				continue
			}
			if curTx != nil {
				curTx.Abort()
				curTx = nil
			}
			inTx = false
			writeTx = false
			fmt.Fprintln(wr, "OK")
		case "CREATE":
			if opts.ReadOnly {
				fmt.Fprintln(wr, "ERR: read-only")
				wr.Flush()
				continue
			}
			if !authed {
				fmt.Fprintln(wr, "ERR: unauthorized")
				wr.Flush()
				continue
			}
			implicit := false
			if !inTx || !writeTx {
				curTx = tm.Begin(false)
				inTx = true
				writeTx = true
				implicit = true
			}
			out, err := eng.Create(cmd.Args[0])
			if err != nil {
				fmt.Fprintln(wr, "ERR:", err)
			} else {
				fmt.Fprintln(wr, out)
			}
			if implicit {
				curTx.Commit()
				curTx = nil
				inTx = false
				writeTx = false
			}
		case "INSERT":
			if opts.ReadOnly {
				fmt.Fprintln(wr, "ERR: read-only")
				wr.Flush()
				continue
			}
			if !authed {
				fmt.Fprintln(wr, "ERR: unauthorized")
				wr.Flush()
				continue
			}
			implicit := false
			if !inTx || !writeTx {
				curTx = tm.Begin(false)
				inTx = true
				writeTx = true
				implicit = true
			}
			out, err := eng.Insert(cmd.Args[0], cmd.Args[1], strings.Join(cmd.Args[2:], " "))
			if err != nil {
				fmt.Fprintln(wr, "ERR:", err)
			} else {
				fmt.Fprintln(wr, out)
			}
			if implicit {
				curTx.Commit()
				curTx = nil
				inTx = false
				writeTx = false
			}
		case "UPDATE":
			if opts.ReadOnly {
				fmt.Fprintln(wr, "ERR: read-only")
				wr.Flush()
				continue
			}
			if !authed {
				fmt.Fprintln(wr, "ERR: unauthorized")
				wr.Flush()
				continue
			}
			implicit := false
			if !inTx || !writeTx {
				curTx = tm.Begin(false)
				inTx = true
				writeTx = true
				implicit = true
			}
			out, err := eng.Update(cmd.Args[0], cmd.Args[1], strings.Join(cmd.Args[2:], " "))
			if err != nil {
				fmt.Fprintln(wr, "ERR:", err)
			} else {
				fmt.Fprintln(wr, out)
			}
			if implicit {
				curTx.Commit()
				curTx = nil
				inTx = false
				writeTx = false
			}
		case "DELETE":
			if opts.ReadOnly {
				fmt.Fprintln(wr, "ERR: read-only")
				wr.Flush()
				continue
			}
			if !authed {
				fmt.Fprintln(wr, "ERR: unauthorized")
				wr.Flush()
				continue
			}
			if len(cmd.Args) == 1 {
				implicit := false
				if !inTx || !writeTx {
					curTx = tm.Begin(false)
					inTx = true
					writeTx = true
					implicit = true
				}
				out, err := eng.Drop(cmd.Args[0])
				if err != nil {
					fmt.Fprintln(wr, "ERR:", err)
				} else {
					fmt.Fprintln(wr, out)
				}
				if implicit {
					curTx.Commit()
					curTx = nil
					inTx = false
					writeTx = false
				}
				break
			}
			implicit := false
			if !inTx || !writeTx {
				curTx = tm.Begin(false)
				inTx = true
				writeTx = true
				implicit = true
			}
			out, err := eng.Delete(cmd.Args[0], cmd.Args[1])
			if err != nil {
				fmt.Fprintln(wr, "ERR:", err)
			} else {
				fmt.Fprintln(wr, out)
			}
			if implicit {
				curTx.Commit()
				curTx = nil
				inTx = false
				writeTx = false
			}
		case "DROP":
			if opts.ReadOnly {
				fmt.Fprintln(wr, "ERR: read-only")
				wr.Flush()
				continue
			}
			if !authed {
				fmt.Fprintln(wr, "ERR: unauthorized")
				wr.Flush()
				continue
			}
			implicit := false
			if !inTx || !writeTx {
				curTx = tm.Begin(false)
				inTx = true
				writeTx = true
				implicit = true
			}
			out, err := eng.Drop(cmd.Args[0])
			if err != nil {
				fmt.Fprintln(wr, "ERR:", err)
			} else {
				fmt.Fprintln(wr, out)
			}
			if implicit {
				curTx.Commit()
				curTx = nil
				inTx = false
				writeTx = false
			}
		case "GET":
			v, err := eng.Get(cmd.Args[0], cmd.Args[1])
			if err != nil {
				fmt.Fprintln(wr, "ERR:", err)
			} else {
				fmt.Fprintln(wr, v)
			}
		case "TABLES":
			for _, n := range eng.ListTables() {
				fmt.Fprintln(wr, n)
			}
		case "SCAN":
			tbl := cmd.Args[0]
			start := ""
			limit := 0
			if len(cmd.Args) >= 2 {
				start = cmd.Args[1]
			}
			if len(cmd.Args) == 3 {
				var L int
				if _, err := fmt.Sscanf(cmd.Args[2], "%d", &L); err == nil {
					limit = L
				}
			}
			pairs, err := eng.Scan(tbl, start, limit)
			if err != nil {
				fmt.Fprintln(wr, "ERR:", err)
			} else {
				for _, kv := range pairs {
					fmt.Fprintf(wr, "%s\t%s\n", kv[0], kv[1])
				}
			}
		case "PREFIXSCAN":
			prefix := cmd.Args[1]
			limit := 0
			if len(cmd.Args) == 3 {
				var L int
				if _, err := fmt.Sscanf(cmd.Args[2], "%d", &L); err == nil {
					limit = L
				}
			}
			pairs, err := eng.PrefixScan(cmd.Args[0], prefix, limit)
			if err != nil {
				fmt.Fprintln(wr, "ERR:", err)
			} else {
				for _, kv := range pairs {
					fmt.Fprintf(wr, "%s\t%s\n", kv[0], kv[1])
				}
			}
		case "EXISTS":
			ok, err := eng.Exists(cmd.Args[0], cmd.Args[1])
			if err != nil {
				fmt.Fprintln(wr, "ERR:", err)
			} else {
				fmt.Fprintln(wr, ok)
			}
		case "RENAME":
			implicit := false
			if !inTx || !writeTx {
				curTx = tm.Begin(false)
				inTx = true
				writeTx = true
				implicit = true
			}
			out, err := eng.Rename(cmd.Args[0], cmd.Args[1])
			if err != nil {
				fmt.Fprintln(wr, "ERR:", err)
			} else {
				fmt.Fprintln(wr, out)
			}
			if implicit {
				curTx.Commit()
				curTx = nil
				inTx = false
				writeTx = false
			}
		case "TRUNCATE":
			implicit := false
			if !inTx || !writeTx {
				curTx = tm.Begin(false)
				inTx = true
				writeTx = true
				implicit = true
			}
			out, err := eng.Truncate(cmd.Args[0])
			if err != nil {
				fmt.Fprintln(wr, "ERR:", err)
			} else {
				fmt.Fprintln(wr, out)
			}
			if implicit {
				curTx.Commit()
				curTx = nil
				inTx = false
				writeTx = false
			}
		case "STATS":
			s, err := eng.Stats(cmd.Args[0])
			if err != nil {
				fmt.Fprintln(wr, "ERR:", err)
			} else {
				fmt.Fprintf(wr, "count=%d height=%d min=%s max=%s\n", s.Count, s.Height, s.MinKey, s.MaxKey)
			}
		case "COUNT":
			n, err := eng.Count(cmd.Args[0])
			if err != nil {
				fmt.Fprintln(wr, "ERR:", err)
			} else {
				fmt.Fprintln(wr, n)
			}
		case "DUMP":
			pairs, err := eng.Scan(cmd.Args[0], "", 0)
			if err != nil {
				fmt.Fprintln(wr, "ERR:", err)
			} else {
				for _, kv := range pairs {
					fmt.Fprintf(wr, "%s\t%s\n", kv[0], kv[1])
				}
			}
		default:
			fmt.Fprintln(wr, "ERR: unknown command")
		}
		wr.Flush()
	}
	if curTx != nil {
		curTx.Abort()
	}
}
