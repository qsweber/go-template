package main

import (
	"encoding/json"
	"net/http"

	"github.com/qsweber/go-template/internal/server"
)

func main() {
	srv := server.New()

	ping := func(w http.ResponseWriter, _ *http.Request) {
		result := srv.Ping()
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]bool{"ok": result}); err != nil {
			http.Error(w, "Error encoding response", http.StatusInternalServerError)
			return
		}
	}

	foo := func(w http.ResponseWriter, r *http.Request) {
		bar := r.URL.Query().Get("bar") // /foo?bar=hello
		input := server.FooInput{Bar: bar}
		output, err := srv.Foo(input)
		if err != nil {
			http.Error(w, "Error calling Foo", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(output); err != nil {
			http.Error(w, "Error encoding response", http.StatusInternalServerError)
			return
		}
	}

	http.HandleFunc("/ping", ping)
	http.HandleFunc("/foo", foo)

	http.ListenAndServe(":8080", nil)
}
