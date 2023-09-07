package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/juliusolson/live/server"
	"golang.org/x/net/websocket"
)

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
