package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"emb/public"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/siuyin/dflt"
	"github.com/starfederation/datastar-go/datastar"
)

var (
	nc *nats.Conn
	ns *server.Server
)

func main() {
	ctx := context.Background()
	ctxStop, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	var err error
	nc, ns, err = embedNATSServer()
	if err != nil {
		log.Println("embed server:", err)
		return
	}
	defer ns.WaitForShutdown()
	defer ns.Shutdown()
	defer nc.Close()

	http.Handle("/", http.FileServerFS(public.Content))
	http.HandleFunc("/send", sendEventsToNATS)
	http.HandleFunc("/clear", clearReceivedArea)

	go func() {
		port := dflt.EnvString("PORT", "8080")
		log.Printf("running web server: PORT=%s", port)
		if err := http.ListenAndServe(":"+port, nil); err != nil {
			log.Println("listen and serve:", err)
			return
		}
	}()

	<-ctxStop.Done()
	log.Println("Interrupt / Terminate signal received")
}

func embedNATSServer() (*nats.Conn, *server.Server, error) {
	opts := &server.Options{Port: 4222, DontListen: true,
		NoSigs: true,
	}

	ns, err := server.NewServer(opts)
	if err != nil {
		return nil, nil, err
	}

	ns.ConfigureLogger()
	ns.Start()
	if !ns.ReadyForConnections(5 * time.Second) {
		return nil, nil, errors.New("could not start embedded NATS server within 5 seconds")
	}

	nc, err := nats.Connect(ns.ClientURL(), nats.InProcessServer(ns))
	if err != nil {
		return nil, nil, err
	}

	return nc, ns, nil
}

type reqSigs struct {
	NumEvents int `json:"numevents"`
}

func sendEventsToNATS(w http.ResponseWriter, r *http.Request) {
	sig := &reqSigs{}
	if err := datastar.ReadSignals(r, sig); err != nil {
		log.Println("sendEvents:", err)
		sse := datastar.NewSSE(w, r)
		sse.PatchElementf(`<p id="received">Error: %v</p>`, err)
		return
	}

	sse := datastar.NewSSE(w, r)
	sse.PatchElementf(`<p id="received">sent %d events.</p>`, sig.NumEvents)

	sub, err := nc.Subscribe("publishdemo", func(m *nats.Msg) {
		s := fmt.Sprintf("<p>event: %s, displayed: %s</p>\n", string(m.Data), time.Now().Format("15:04:05.000000"))
		if err := sse.PatchElements(s, datastar.WithSelectorID("received"), datastar.WithModeAppend()); err != nil {
			log.Println("patch elements:", err)
			return
		}
	})
	if err != nil {
		log.Println("subscribe", err)
		sse.PatchElementf(`<p id="received">%v</p>`, err)
		return
	}
	defer func() { // wait for the drain to complete
		for {
			if !sub.IsDraining() {
				break
			}
			time.Sleep(time.Millisecond)
		}
	}()
	defer sub.Drain()

	for i := 0; i < sig.NumEvents; i++ {
		nc.Publish("publishdemo", []byte(fmt.Sprintf(`%d: %s`, i+1, time.Now().Format("2006-01-02 15:04:05.000000 -0700"))))
	}
}

func clearReceivedArea(w http.ResponseWriter, r *http.Request) {
	sse := datastar.NewSSE(w, r)
	sse.PatchElements(`<p id="received">cleared ...</p>`)
}
