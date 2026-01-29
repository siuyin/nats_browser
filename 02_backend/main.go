package main

import (
	"bkend/public"
	"fmt"
	"log"
	"net/http"

	"github.com/nats-io/nats.go"
	"github.com/siuyin/dflt"
)

func main() {
	http.Handle("/", http.FileServer(http.FS(public.Content)))
	http.HandleFunc("/connect", connectToNATS)

	port := dflt.EnvString("PORT", "8080")
	url := dflt.EnvString("NATS_URL", "yourToken@yourNATSServer.com")
	log.Printf("running web server: PORT=%s, NATS_URL=%s...", port, url[:9])
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func connectToNATS(w http.ResponseWriter, r *http.Request) {
	url := dflt.EnvString("NATS_URL", "yourToken@yourNATSServer.com")
	nc, err := nats.Connect(url)
	if err != nil {
		log.Println("connectToNATS:", err)
		return
	}

	defer nc.Close()

	fmt.Fprintf(w, "Connected to NATS servers: %v\n", nc.Servers())
}
