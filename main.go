package main

import (
	"net/http"
	"os"

	"github.com/gorilla/handlers"
	"github.com/micro/services/url/proxy"
)

func main() {
	// logging handler
	handler := handlers.LoggingHandler(os.Stdout, http.HandlerFunc(proxy.Handler))
	// register the proxy handler
	http.Handle("/", handler)
	// run on port 8080
	http.ListenAndServe(":8080", nil)
}
