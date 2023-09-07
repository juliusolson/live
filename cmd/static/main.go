package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
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

	args := os.Args[1:]
	var dir string
	var port int
	var err error

	switch len(args) {
	case 0:
		dir = "."
		port = 8080
	case 1:
		port, err = strconv.Atoi(args[0])
		if err != nil {
			port = 8080
			dir = args[0]
		} else {
			dir = "."
		}
	case 2:
		dir = args[0]
		port, err = strconv.Atoi(args[1])
		if err != nil {
			log.Fatalf("%v not an int", args[1])
		}
	}
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
