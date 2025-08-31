package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"sharkDB/internal/engine"
	"sharkDB/internal/httpserver"
	"sharkDB/internal/pager2"
	"sharkDB/internal/parser"
	"sharkDB/internal/server"
	"sharkDB/internal/txn"
)

func main() {
	dbFlag := flag.String("db", "sharkdb.gob", "path to database file")
	serve := flag.String("serve", "", "listen address for TCP server, e.g. :8080 (empty = CLI mode)")
	httpAddr := flag.String("http", "", "listen address for HTTP server, e.g. :8090 (empty = off)")
	auth := flag.String("auth", "", "require this token for TCP writes (AUTH <token>)")
	readonly := flag.Bool("readonly", false, "start TCP server in read-only mode (blocks writes)")
	httpAuth := flag.String("httpauth", "", "require this bearer token for HTTP writes")
	httpReadonly := flag.Bool("httpreadonly", false, "start HTTP server in read-only mode (blocks writes)")
	flag.Parse()

	dbPath := *dbFlag
	p, err := pager2.Open(dbPath)
	if err != nil {
		log.Fatalf("open pager: %v", err)
	}
	eng := engine.New(p)
	tm := txn.NewManager()

	if *serve != "" {
		log.Printf("starting server on %s", *serve)
		opts := server.Options{RequireToken: *auth, ReadOnly: *readonly}
		if err := server.Serve(*serve, eng, tm, opts); err != nil {
			log.Fatal(err)
		}
		return
	}

	if *httpAddr != "" {
		log.Printf("starting HTTP server on %s", *httpAddr)
		if err := httpserver.Start(*httpAddr, eng, tm, httpserver.Options{RequireToken: *httpAuth, ReadOnly: *httpReadonly}); err != nil {
			log.Fatal(err)
		}
		return
	}

	fmt.Println("sharkDB ready. Commands: CREATE/INSERT/GET/UPDATE/DELETE/BEGIN/COMMIT/ABORT. Ctrl+C to exit.")
	in := bufio.NewScanner(os.Stdin)
	var inTx bool
	var writeTx bool
	var curTx *txn.Tx
	for {
		if inTx {
			fmt.Print("sharkdb(tx)> ")
		} else {
			fmt.Print("sharkdb> ")
		}
		if !in.Scan() {
			break
		}
		line := strings.TrimSpace(in.Text())
		if line == "" {
			continue
		}
		cmd, err := parser.Parse(line)
		if err != nil {
			fmt.Println("ERR:", err)
			continue
		}

		switch cmd.Name {
		case "HELP":
			fmt.Println("Commands:")
			fmt.Println("  BEGIN [READONLY] | COMMIT | ABORT")
			fmt.Println("  CREATE <table> | DROP <table> | RENAME <old> <new> | TRUNCATE <table>")
			fmt.Println("  INSERT <table> <key> <value> | UPDATE <table> <key> <value> | DELETE <table> [key]")
			fmt.Println("  GET <table> <key> | EXISTS <table> <key>")
			fmt.Println("  TABLES | SCAN <table> [start] [limit] | PREFIXSCAN <table> <prefix> [limit]")
			fmt.Println("  COUNT <table> | STATS <table> | DUMP <table> [file] | LOAD <table> <file>")
			fmt.Println("  HELP | EXIT | QUIT")
			continue
		case "EXIT", "QUIT":
			fmt.Println("Bye")
			return
		case "BEGIN":
			if inTx {
				fmt.Println("ERR: already in transaction")
				continue
			}
			readOnly := len(cmd.Args) == 1 && cmd.Args[0] == "READONLY"
			// Acquire write lock if not readonly
			if !readOnly {
				curTx = tm.Begin(false)
			} else {
				curTx = nil // no lock needed for readonly
			}
			inTx = true
			writeTx = !readOnly
			fmt.Println("OK")
		case "COMMIT":
			if !inTx {
				fmt.Println("ERR: not in transaction")
				continue
			}
			if curTx != nil {
				curTx.Commit()
				curTx = nil
			}
			inTx = false
			writeTx = false
			fmt.Println("OK")
		case "ABORT":
			if !inTx {
				fmt.Println("ERR: not in transaction")
				continue
			}
			if curTx != nil {
				curTx.Abort()
				curTx = nil
			}
			inTx = false
			writeTx = false
			fmt.Println("OK")
		case "CREATE":
			implicit := false
			if !inTx || !writeTx {
				curTx = tm.Begin(false)
				inTx = true
				writeTx = true
				implicit = true
			}
			table := cmd.Args[0]
			if out, err := eng.Create(table); err != nil {
				fmt.Println("ERR:", err)
			} else {
				fmt.Println(out)
			}
			if implicit {
				curTx.Commit()
				curTx = nil
				inTx = false
				writeTx = false
			}
		case "INSERT":
			implicit := false
			if !inTx || !writeTx {
				curTx = tm.Begin(false)
				inTx = true
				writeTx = true
				implicit = true
			}
			table, key, val := cmd.Args[0], cmd.Args[1], cmd.Args[2]
			if out, err := eng.Insert(table, key, val); err != nil {
				fmt.Println("ERR:", err)
			} else {
				fmt.Println(out)
			}
			if implicit {
				curTx.Commit()
				curTx = nil
				inTx = false
				writeTx = false
			}
		case "UPDATE":
			implicit := false
			if !inTx || !writeTx {
				curTx = tm.Begin(false)
				inTx = true
				writeTx = true
				implicit = true
			}
			table, key, val := cmd.Args[0], cmd.Args[1], cmd.Args[2]
			if out, err := eng.Update(table, key, val); err != nil {
				fmt.Println("ERR:", err)
			} else {
				fmt.Println(out)
			}
			if implicit {
				curTx.Commit()
				curTx = nil
				inTx = false
				writeTx = false
			}
		case "DELETE":
			if len(cmd.Args) == 1 {
				// DELETE <table> : drop table shorthand (allow implicit tx)
				implicit := false
				if !inTx || !writeTx {
					curTx = tm.Begin(false)
					inTx = true
					writeTx = true
					implicit = true
				}
				table := cmd.Args[0]
				if out, err := eng.Drop(table); err != nil {
					fmt.Println("ERR:", err)
				} else {
					fmt.Println(out)
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
			table, key := cmd.Args[0], cmd.Args[1]
			if out, err := eng.Delete(table, key); err != nil {
				fmt.Println("ERR:", err)
			} else {
				fmt.Println(out)
			}
			if implicit {
				curTx.Commit()
				curTx = nil
				inTx = false
				writeTx = false
			}
		case "DROP":
			implicit := false
			if !inTx || !writeTx {
				curTx = tm.Begin(false)
				inTx = true
				writeTx = true
				implicit = true
			}
			table := cmd.Args[0]
			if out, err := eng.Drop(table); err != nil {
				fmt.Println("ERR:", err)
			} else {
				fmt.Println(out)
			}
			if implicit {
				curTx.Commit()
				curTx = nil
				inTx = false
				writeTx = false
			}
		case "GET":
			table, key := cmd.Args[0], cmd.Args[1]
			if v, err := eng.Get(table, key); err != nil {
				fmt.Println("ERR:", err)
			} else {
				fmt.Println(v)
			}
		case "TABLES":
			names := eng.ListTables()
			for _, n := range names {
				fmt.Println(n)
			}
		case "SCAN":
			if len(cmd.Args) == 0 {
				fmt.Println("ERR: SCAN requires at least <table>")
				continue
			}
			tbl := cmd.Args[0]
			start := ""
			limit := 0
			if len(cmd.Args) >= 2 {
				start = cmd.Args[1]
			}
			if len(cmd.Args) == 3 {
				// parse limit
				var L int
				_, err := fmt.Sscanf(cmd.Args[2], "%d", &L)
				if err != nil {
					fmt.Println("ERR: bad limit")
					continue
				}
				limit = L
			}
			pairs, err := eng.Scan(tbl, start, limit)
			if err != nil {
				fmt.Println("ERR:", err)
				continue
			}
			for _, kv := range pairs {
				fmt.Printf("%s\t%s\n", kv[0], kv[1])
			}
		case "PREFIXSCAN":
			if len(cmd.Args) < 2 || len(cmd.Args) > 3 {
				fmt.Println("ERR: PREFIXSCAN <table> <prefix> [limit]")
				continue
			}
			tbl := cmd.Args[0]
			prefix := cmd.Args[1]
			limit := 0
			if len(cmd.Args) == 3 {
				var L int
				if _, err := fmt.Sscanf(cmd.Args[2], "%d", &L); err == nil {
					limit = L
				}
			}
			pairs, err := eng.PrefixScan(tbl, prefix, limit)
			if err != nil {
				fmt.Println("ERR:", err)
				continue
			}
			for _, kv := range pairs {
				fmt.Printf("%s\t%s\n", kv[0], kv[1])
			}
		case "EXISTS":
			if len(cmd.Args) != 2 {
				fmt.Println("ERR: EXISTS <table> <key>")
				continue
			}
			ok, err := eng.Exists(cmd.Args[0], cmd.Args[1])
			if err != nil {
				fmt.Println("ERR:", err)
				continue
			}
			fmt.Println(ok)
		case "RENAME":
			if len(cmd.Args) != 2 {
				fmt.Println("ERR: RENAME <old> <new>")
				continue
			}
			implicit := false
			if !inTx || !writeTx {
				curTx = tm.Begin(false)
				inTx = true
				writeTx = true
				implicit = true
			}
			if out, err := eng.Rename(cmd.Args[0], cmd.Args[1]); err != nil {
				fmt.Println("ERR:", err)
			} else {
				fmt.Println(out)
			}
			if implicit {
				curTx.Commit()
				curTx = nil
				inTx = false
				writeTx = false
			}
		case "TRUNCATE":
			if len(cmd.Args) != 1 {
				fmt.Println("ERR: TRUNCATE <table>")
				continue
			}
			implicit := false
			if !inTx || !writeTx {
				curTx = tm.Begin(false)
				inTx = true
				writeTx = true
				implicit = true
			}
			if out, err := eng.Truncate(cmd.Args[0]); err != nil {
				fmt.Println("ERR:", err)
			} else {
				fmt.Println(out)
			}
			if implicit {
				curTx.Commit()
				curTx = nil
				inTx = false
				writeTx = false
			}
		case "STATS":
			if len(cmd.Args) != 1 {
				fmt.Println("ERR: STATS <table>")
				continue
			}
			s, err := eng.Stats(cmd.Args[0])
			if err != nil {
				fmt.Println("ERR:", err)
				continue
			}
			fmt.Printf("count=%d height=%d min=%s max=%s\n", s.Count, s.Height, s.MinKey, s.MaxKey)
		case "COUNT":
			if len(cmd.Args) != 1 {
				fmt.Println("ERR: COUNT <table>")
				continue
			}
			tbl := cmd.Args[0]
			n, err := eng.Count(tbl)
			if err != nil {
				fmt.Println("ERR:", err)
				continue
			}
			fmt.Println(n)
		case "DUMP":
			// DUMP <table> [filepath]; prints TSV if no file
			if len(cmd.Args) != 1 && len(cmd.Args) != 2 {
				fmt.Println("ERR: DUMP <table> [file]")
				continue
			}
			tbl := cmd.Args[0]
			pairs, err := eng.Scan(tbl, "", 0)
			if err != nil {
				fmt.Println("ERR:", err)
				continue
			}
			if len(cmd.Args) == 1 {
				for _, kv := range pairs {
					fmt.Printf("%s\t%s\n", kv[0], kv[1])
				}
			} else {
				f, err := os.Create(cmd.Args[1])
				if err != nil {
					fmt.Println("ERR:", err)
					continue
				}
				w := bufio.NewWriter(f)
				for _, kv := range pairs {
					fmt.Fprintf(w, "%s\t%s\n", kv[0], kv[1])
				}
				w.Flush()
				f.Close()
				fmt.Println("OK")
			}
		case "LOAD":
			// LOAD <table> <file> (TSV: key\tvalue)
			if len(cmd.Args) != 2 {
				fmt.Println("ERR: LOAD <table> <file>")
				continue
			}
			implicit := false
			if !inTx || !writeTx {
				curTx = tm.Begin(false)
				inTx = true
				writeTx = true
				implicit = true
			}
			tbl, path := cmd.Args[0], cmd.Args[1]
			f, err := os.Open(path)
			if err != nil {
				fmt.Println("ERR:", err)
				if implicit {
					curTx.Abort()
					curTx = nil
					inTx = false
					writeTx = false
				}
				continue
			}
			sc := bufio.NewScanner(f)
			for sc.Scan() {
				line := sc.Text()
				if line == "" {
					continue
				}
				var k, v string
				parts := strings.SplitN(line, "\t", 2)
				if len(parts) != 2 {
					fmt.Println("ERR: bad line", line)
					if implicit {
						curTx.Abort()
						curTx = nil
						inTx = false
						writeTx = false
					}
					f.Close()
					continue
				}
				k, v = parts[0], parts[1]
				if _, err := eng.Insert(tbl, k, v); err != nil {
					fmt.Println("ERR:", err)
					if implicit {
						curTx.Abort()
						curTx = nil
						inTx = false
						writeTx = false
					}
					f.Close()
					continue
				}
			}
			f.Close()
			if implicit {
				curTx.Commit()
				curTx = nil
				inTx = false
				writeTx = false
			}
			fmt.Println("OK")
		default:
			fmt.Println("ERR: unknown command")
		}
	}
}
