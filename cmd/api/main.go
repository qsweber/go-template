package main

import (
	"context"
	"net/http"

	"github.com/qsweber/go-template/internal/auth"
	"github.com/qsweber/go-template/internal/rpc"
	"github.com/qsweber/go-template/internal/server"
)

type mockVerifier struct{}

func (m mockVerifier) VerifyToken(ctx context.Context, tokenString string) (*auth.Claims, error) {
	return &auth.Claims{}, nil
}

func main() {
	srv := server.New()
	tokenVerifier := mockVerifier{}

	genericHandler := func(w http.ResponseWriter, r *http.Request) {
		headers := make(map[string]string)
		for k, v := range r.Header {
			if len(v) > 0 {
				headers[k] = v[0]
			}
		}
		result := rpc.Handler(
			r.Context(),
			rpc.Request{
				Path:    r.URL.Path,
				Headers: headers,
			},
			srv,
			tokenVerifier,
		)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(result.StatusCode)
		if _, err := w.Write([]byte(result.Body)); err != nil {
			return
		}
	}

	http.HandleFunc("/ping", genericHandler)
	http.HandleFunc("/foo", genericHandler)

	http.ListenAndServe(":8080", nil)
}
