// Command api runs the TrustCheck verification API as a long-running HTTP
// server on $PORT (default 8080). This is the entry point for local
// development; the serverless entry point for Netlify lives in
// function/api/main.go.
package main

import (
	"os"

	"github.com/pamierin/trustcheck/apps/api/internal/server"
)

func main() {
	addr := os.Getenv("PORT")
	if addr == "" {
		addr = "8080"
	}
	server.NewRouter("").Run(":" + addr)
}
