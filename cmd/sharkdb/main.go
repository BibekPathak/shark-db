package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"sharkDB/internal/engine"
	"sharkDB/internal/pager"
	"sharkDB/internal/parser"
	"sharkDB/internal/txn"
)

func main() {
	dbPath := "sharkdb.gob"
	p, err := pager.Open(dbPath)
	if err != nil {
		log.Fatalf("open pager: %v", err)
	}
	eng := engine.New(p)
	tm := txn.NewManager()

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
			if !inTx || !writeTx {
				fmt.Println("ERR: must be in write transaction")
				continue
			}
			table := cmd.Args[0]
			if out, err := eng.Create(table); err != nil {
				fmt.Println("ERR:", err)
			} else {
				fmt.Println(out)
			}
		case "INSERT":
			if !inTx || !writeTx {
				fmt.Println("ERR: must be in write transaction")
				continue
			}
			table, key, val := cmd.Args[0], cmd.Args[1], cmd.Args[2]
			if out, err := eng.Insert(table, key, val); err != nil {
				fmt.Println("ERR:", err)
			} else {
				fmt.Println(out)
			}
		case "UPDATE":
			if !inTx || !writeTx {
				fmt.Println("ERR: must be in write transaction")
				continue
			}
			table, key, val := cmd.Args[0], cmd.Args[1], cmd.Args[2]
			if out, err := eng.Update(table, key, val); err != nil {
				fmt.Println("ERR:", err)
			} else {
				fmt.Println(out)
			}
		case "DELETE":
			if len(cmd.Args) == 1 {
				// DELETE <table> : drop table shorthand
				if !inTx || !writeTx {
					fmt.Println("ERR: must be in write transaction")
					continue
				}
				table := cmd.Args[0]
				if out, err := eng.Drop(table); err != nil {
					fmt.Println("ERR:", err)
				} else {
					fmt.Println(out)
				}
				break
			}
			if !inTx || !writeTx {
				fmt.Println("ERR: must be in write transaction")
				continue
			}
			table, key := cmd.Args[0], cmd.Args[1]
			if out, err := eng.Delete(table, key); err != nil {
				fmt.Println("ERR:", err)
			} else {
				fmt.Println(out)
			}
		case "DROP":
			if !inTx || !writeTx {
				fmt.Println("ERR: must be in write transaction")
				continue
			}
			table := cmd.Args[0]
			if out, err := eng.Drop(table); err != nil {
				fmt.Println("ERR:", err)
			} else {
				fmt.Println(out)
			}
		case "GET":
			table, key := cmd.Args[0], cmd.Args[1]
			if v, err := eng.Get(table, key); err != nil {
				fmt.Println("ERR:", err)
			} else {
				fmt.Println(v)
			}
		default:
			fmt.Println("ERR: unknown command")
		}
	}
}
