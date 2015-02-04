// +build !appengine

package main

import (
	"flag"
	"net/http"
)

func main() {
	port := flag.String("http", "127.0.0.1:8080", "ip and port to listen to")
	flag.Parse()
	http.HandleFunc("/", homeHandler)
	http.ListenAndServe(*port, nil)
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "static/home.html")
}
