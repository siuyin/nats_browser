package main

import (
	"fmt"
	"log"
	"net/http"
	"strm/public"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/siuyin/dflt"
	"github.com/starfederation/datastar-go/datastar"
)

var nc *nats.Conn

func main() {
	setupNATSConnection()
	defer nc.Close()

	http.Handle("/", http.FileServerFS(public.Content))
	http.HandleFunc("/send", sendEventsToNATS)
	http.HandleFunc("/clear", clearReceivedArea)

	port := dflt.EnvString("PORT", "8080")
	url := dflt.EnvString("NATS_URL", "yourToken@yourNATSServer.com")
	log.Printf("running web server: PORT=%s, NATS_URL=%s...", port, url[:9])
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Println(err)
		return
	}
}

func setupNATSConnection() {
	var err error
	url := dflt.EnvString("NATS_URL", "yourToken@yourNATSServer.com")
	nc, err = nats.Connect(url)
	if err != nil {
		log.Fatal("setupNATSConnection:", err)
	}
}

type reqSigs struct {
	NumEvents int `json:"numevents"`
}

func sendEventsToNATS(w http.ResponseWriter, r *http.Request) {
	sig := &reqSigs{}
	if err := datastar.ReadSignals(r, sig); err != nil {
		log.Println("sendEvents:", err)
		sse := datastar.NewSSE(w, r)
		sse.PatchElements(fmt.Sprintf(`<p id="received">Error: %v</p>`, err))
		return
	}

	sse := datastar.NewSSE(w, r)
	sse.PatchElements(fmt.Sprintf(`<p id="received">sent %d events.</p>`, sig.NumEvents))

	sub, err := nc.Subscribe("publishdemo", func(m *nats.Msg) {
		s := fmt.Sprintf("<p>event: %s, displayed: %s</p>\n", string(m.Data), time.Now().Format("15:04:05.000000"))
		if err := sse.PatchElements(s, datastar.WithSelectorID("received"), datastar.WithModeAppend()); err != nil {
			log.Println("patch elements:", err)
			return
		}
	})
	if err != nil {
		log.Println("subscribe", err)
		sse.PatchElements(fmt.Sprintf(`<p id="received">%v</p>`, err))
		return
	}

	for i := 0; i < sig.NumEvents; i++ {
		nc.Publish("publishdemo", []byte(fmt.Sprintf(`%d: %s`, i+1, time.Now().Format("2006-01-02 15:04:05.000000 -0700"))))

	}

	sub.Drain()
	for {
		if !sub.IsDraining() {
			break
		}
		time.Sleep(time.Millisecond)
	}

}

func clearReceivedArea(w http.ResponseWriter, r *http.Request) {
	sse := datastar.NewSSE(w, r)
	sse.PatchElements(`<p id="received">cleared ...</p>`)
}
