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
	s := &LiveServer{
		ActiveWS: nil,
		Dir:      dir,
		Port:     port,
	}
	http.HandleFunc("/", s.Static)
	http.Handle("/ws", websocket.Handler(s.HandleWS))
	return s
}

func (s *LiveServer) Listen() error {
	// err := exec.Command("open", fmt.Sprintf("http://localhost:%v", s.Port)).Start()
	// if err != nil {
	// 	log.Fatal(err)
	// }
	fmt.Printf("Event log:\n==========\n")
	return http.ListenAndServe(fmt.Sprintf(":%v", s.Port), nil)
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

	// Add the root and all sub dirs.
	err = filepath.Walk(s.Dir, func(fp string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if fi.IsDir() {
			err = watcher.Add(fp)
			if err != nil {
				return err
			}
		}
		return nil
	})

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
ws.onclose = (event) => {
    let b=document.querySelector("body");
    let d = document.createElement("div");
    d.style.cssText="position:absolute;top:0px;left:0px;width:100%%;height:100%%;background:black;opacity:0.85;display:flex;align-items:center;justify-content:center;color:white;font-size:2em;font-family:monospace;";
    d.textContent="Live Server Disconnected";
    b.appendChild(d);
}
</script>

    `, port)

	return s[:oglog[0]] + js + s[oglog[0]:]
}

func (s *LiveServer) Static(w http.ResponseWriter, r *http.Request) {
	indexPage := "index.html"

	fp := filepath.Join(s.Dir, r.URL.Path)
	stat, err := os.Stat(fp)
	if err != nil {
		log.Fatal(err)
	}
	if stat.IsDir() {
		fp = filepath.Join(fp, indexPage)
	}

	fmt.Println(r.URL.Path, fp)

	// TODO: add astro, vue, svelte etc
	if !strings.HasSuffix(fp, "html") {
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
