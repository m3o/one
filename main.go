package main

import (
	"net/http"

	"github.com/micro/services/url/proxy"
)

func main() {
	// register the proxy handler
	http.HandleFunc("/", proxy.Handler)
	// run on port 8080
	http.ListenAndServe(":8080", nil)
}
