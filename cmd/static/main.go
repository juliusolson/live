package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

func logStuff(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path

		if p == "/" {
			p = "/index.html"
		}

		log.Printf("Serving %v", p)

		h.ServeHTTP(w, r)
	})
}

func main() {

	var port int
	var dir string

	flag.IntVar(&port, "port", 8080, "port to bind the server to")
	flag.StringVar(&dir, "dir", ".", "which directory to serve")
	flag.Parse()

	fs := http.FileServer(http.Dir(dir))
	http.Handle("/", logStuff(fs))

	abspath, err := filepath.Abs(dir)
	if err != nil {
		log.Fatalf("'%v' is not a valid path", dir)
	}

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		log.Fatalf("path '%v' does not exist", dir)
	}

	fmt.Printf("Serving %v on %v\n\n", abspath, port)
	fmt.Printf("Event log:\n==========\n")
	http.ListenAndServe(fmt.Sprintf(":%v", port), nil)

}
