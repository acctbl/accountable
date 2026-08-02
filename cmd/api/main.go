package main

import (
	"log"
	"net/http"

	"github.com/acctbl/accountable/gen/go/accountable/system/v1/systemv1connect"
	"github.com/acctbl/accountable/internal/server"
)

func main() {
	mux := http.NewServeMux()

	path, handler := systemv1connect.NewSystemServiceHandler(&server.SystemServer{})
	mux.Handle(path, handler)

	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
