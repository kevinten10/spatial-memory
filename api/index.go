package handler

import (
	"net/http"

	"github.com/spatial-memory/spatial-memory/bridge"
)

func init() {
	bridge.InitApp()
}

// Handler is the Vercel serverless function entry point
func Handler(w http.ResponseWriter, r *http.Request) {
	bridge.GinEngine.ServeHTTP(w, r)
}
