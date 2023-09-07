package server

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"golang.org/x/net/websocket"
)

type LiveServer struct {
	ActiveWS *websocket.Conn
	Dir      string
	Port     int
}

func New(dir string, port int) *LiveServer {
	return &LiveServer{
		ActiveWS: nil,
		Dir:      dir,
		Port:     port,
	}
}

func (s *LiveServer) WatchDir() {
	// Create new watcher.
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	// Start listening for events.
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Has(fsnotify.Create) || event.Has(fsnotify.Write) {
					if strings.HasSuffix(event.Name, "~") {
						continue
					}
					log.Printf("change detected in file: %v", event.Name)
					if s.ActiveWS != nil {
						s.ActiveWS.Write([]byte("reload"))
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("error:", err)
			}
		}
	}()

	// Add a path.
	err = watcher.Add(s.Dir)
	if err != nil {
		log.Fatal(err)
	}
	select {}
}

func (s *LiveServer) HandleWS(ws *websocket.Conn) {
	// This is now the active WS
	s.ActiveWS = ws

	open := true
	msg := make([]byte, 0)
	// Read until close.
	for open {
		_, err := ws.Read(msg)
		if err == io.EOF {
			open = false
		}
		time.Sleep(100 * time.Millisecond)
	}

	// If no new WS connection - clean up
	if s.ActiveWS == ws {
		s.ActiveWS = nil
	}

	// Clean up
	err := ws.Close()
	if err != nil {
		log.Println(err)
	}
}

func injectSocketReload(s string, port int) string {
	rx, err := regexp.Compile("</body>")
	if err != nil {
		log.Fatal(err)
	}

	oglog := rx.FindStringIndex(s)

	if len(oglog) < 1 {
		return s
	}

	js := fmt.Sprintf(`
<script>
let ws = new WebSocket("ws://localhost:%v/ws");
ws.onmessage = (event) => {window.location.reload(true)}
</script>

    `, port)

	return s[:oglog[0]] + js + s[oglog[0]:]
}

func (s *LiveServer) Static(w http.ResponseWriter, r *http.Request) {

	path := r.URL.Path
	if path == "/" {
		path += "index.html"
	}

	fp := filepath.Join(s.Dir, path)

	fmt.Println(path, fp)

	// TODO: add astro, vue, svelte etc
	if !strings.HasSuffix(path, "html") {
		http.ServeFile(w, r, fp)
		return
	}

	f, err := os.Open(fp)
	if err != nil {
		log.Fatal(err)
	}
	b, _ := io.ReadAll(f)
	str := injectSocketReload(string(b), s.Port)
	fmt.Fprint(w, str)
}
