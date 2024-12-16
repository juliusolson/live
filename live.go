package live

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
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
	http.Handle("/", s.injector(http.FileServer(http.Dir(dir))))
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
	ogloc := strings.Index(s, "</body>")
	if ogloc < 0 {
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

	return s[:ogloc] + js + s[ogloc:]
}

func (s *LiveServer) injector(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fp := filepath.Join(s.Dir, r.URL.Path)

		// If not exists -> 404
		fpStat, err := os.Stat(fp)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		// If not dir and not html
		if !fpStat.IsDir() && !strings.HasSuffix(r.URL.Path, ".html") {
			next.ServeHTTP(w, r)
			return
		}

		// Dir and no index file
		_, indexErr := os.Stat(filepath.Join(fp, "index.html"))
		if fpStat.IsDir() && indexErr != nil {
			next.ServeHTTP(w, r)
			return
		}

		// Finally add index.html if dir
		if fpStat.IsDir() && indexErr == nil {
			fp = filepath.Join(fp, "index.html")
		}

		// read, inject and serve
		f, err := os.Open(fp)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		b, _ := io.ReadAll(f)
		str := injectSocketReload(string(b), s.Port)
		_, err = fmt.Fprint(w, str)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

	})
}
