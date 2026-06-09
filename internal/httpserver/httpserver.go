package httpserver

import (
	"io"
	"net/http"
	"strconv"

	"sharkDB/internal/engine"
	"sharkDB/internal/txn"
)

type Options struct {
	RequireToken string
	ReadOnly     bool
}

func Start(addr string, eng *engine.Engine, tm *txn.Manager, opts Options) error {
	mux := http.NewServeMux()

	mux.HandleFunc("/tables", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			for _, n := range eng.ListTables() {
				_, _ = io.WriteString(w, n+"\n")
			}
			return
		}
		if r.Method == http.MethodPost {
			if opts.ReadOnly {
				http.Error(w, "read-only", http.StatusForbidden)
				return
			}
			if opts.RequireToken != "" && r.Header.Get("Authorization") != "Bearer "+opts.RequireToken {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			tbl := r.URL.Query().Get("name")
			if tbl == "" {
				b, _ := io.ReadAll(r.Body)
				tbl = string(b)
			}
			tx := tm.Begin(false)
			defer tx.Commit()
			if out, err := eng.Create(tbl); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
			} else {
				_, _ = io.WriteString(w, out+"\n")
			}
			return
		}
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	})

	mux.HandleFunc("/tables/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if opts.ReadOnly {
			http.Error(w, "read-only", http.StatusForbidden)
			return
		}
		if opts.RequireToken != "" && r.Header.Get("Authorization") != "Bearer "+opts.RequireToken {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		name := r.URL.Path[len("/tables/"):]
		if name == "" {
			http.Error(w, "missing table", http.StatusBadRequest)
			return
		}
		tx := tm.Begin(false)
		defer tx.Commit()
		if _, err := eng.Drop(name); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		_, _ = io.WriteString(w, "OK\n")
	})

	mux.HandleFunc("/kv/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path[len("/kv/"):]
		slash := -1
		for i := 0; i < len(path); i++ {
			if path[i] == '/' {
				slash = i
				break
			}
		}
		if slash == -1 {
			http.Error(w, "expected /kv/{table}/{key}", http.StatusBadRequest)
			return
		}
		table := path[:slash]
		key := path[slash+1:]
		switch r.Method {
		case http.MethodGet:
			tx := tm.Begin(true)
			defer tx.Abort()
			v, err := eng.Get(table, key, tx)
			if err != nil {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			_, _ = io.WriteString(w, v)
		case http.MethodPut:
			if opts.ReadOnly {
				http.Error(w, "read-only", http.StatusForbidden)
				return
			}
			if opts.RequireToken != "" && r.Header.Get("Authorization") != "Bearer "+opts.RequireToken {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			b, _ := io.ReadAll(r.Body)
			tx := tm.Begin(false)
			defer tx.Commit()
			if _, err := eng.Update(table, key, string(b), tx); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			_, _ = io.WriteString(w, "OK\n")
		case http.MethodDelete:
			if opts.ReadOnly {
				http.Error(w, "read-only", http.StatusForbidden)
				return
			}
			if opts.RequireToken != "" && r.Header.Get("Authorization") != "Bearer "+opts.RequireToken {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			tx := tm.Begin(false)
			defer tx.Commit()
			if _, err := eng.Delete(table, key, tx); err != nil {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			_, _ = io.WriteString(w, "OK\n")
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/scan/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		table := r.URL.Path[len("/scan/"):]
		start := r.URL.Query().Get("start")
		limit := 0
		if s := r.URL.Query().Get("limit"); s != "" {
			if n, err := strconv.Atoi(s); err == nil {
				limit = n
			}
		}
		tx := tm.Begin(true)
		defer tx.Abort()
		pairs, err := eng.Scan(table, start, limit, tx)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		for _, kv := range pairs {
			_, _ = io.WriteString(w, kv[0]+"\t"+kv[1]+"\n")
		}
	})

	mux.HandleFunc("/prefix/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		table := r.URL.Path[len("/prefix/"):]
		prefix := r.URL.Query().Get("prefix")
		limit := 0
		if s := r.URL.Query().Get("limit"); s != "" {
			if n, err := strconv.Atoi(s); err == nil {
				limit = n
			}
		}
		tx := tm.Begin(true)
		defer tx.Abort()
		pairs, err := eng.PrefixScan(table, prefix, limit, tx)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		for _, kv := range pairs {
			_, _ = io.WriteString(w, kv[0]+"\t"+kv[1]+"\n")
		}
	})

	mux.HandleFunc("/stats/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		table := r.URL.Path[len("/stats/"):]
		tx := tm.Begin(true)
		defer tx.Abort()
		s, err := eng.Stats(table, tx)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		_, _ = io.WriteString(w, "count="+strconv.Itoa(s.Count)+" height="+strconv.Itoa(s.Height)+" min="+s.MinKey+" max="+s.MaxKey+"\n")
	})

	return http.ListenAndServe(addr, mux)
}
