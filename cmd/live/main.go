package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/juliusolson/live/server"
	"golang.org/x/net/websocket"
)

func main() {

	var port int
	var dir string

	flag.IntVar(&port, "port", 8080, "port to bind the server to")
	flag.StringVar(&dir, "dir", ".", "which directory to serve")
	flag.Parse()

	abspath, err := filepath.Abs(dir)
	if err != nil {
		log.Fatalf("'%v' is not a valid path", dir)
	}

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		log.Fatalf("path '%v' does not exist", dir)
	}

	s := server.New(dir, port)
	go s.WatchDir()

	fmt.Printf("Serving %v on %v\n\n", abspath, port)
	fmt.Printf("Event log:\n==========\n")

	http.HandleFunc("/", s.Static)
	http.Handle("/ws", websocket.Handler(s.HandleWS))
	http.ListenAndServe(fmt.Sprintf(":%v", port), nil)
}
