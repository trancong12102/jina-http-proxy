package main

import (
	"net/http"

	"github.com/trancong12102/jina-http-proxy/key"
)

func createApiRouter(
	keyHandler *key.KeyHandler,
) http.Handler {
	router := http.NewServeMux()

	// Key
	router.HandleFunc("GET /keys/stats", keyHandler.GetKeyStats)
	router.HandleFunc("POST /keys", keyHandler.InsertKey)

	return router
}
