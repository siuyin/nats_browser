package main

import (
	"bkend/public"
	"fmt"
	"log"
	"net/http"

	"github.com/nats-io/nats.go"
	"github.com/siuyin/dflt"
)

var nc *nats.Conn

func main() {
	setupNATSConnection()
	defer nc.Close()

	http.Handle("/", http.FileServer(http.FS(public.Content)))
	http.HandleFunc("/connect", connectToNATS)

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

func connectToNATS(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Connected to NATS servers: %v\n", nc.Servers())
}
