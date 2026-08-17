package main

import (
	"log"
	"net/http"
)

func main(){
	const filepathRoot = "."
	const port = "8080"

	srvMux := http.NewServeMux()
	srvMux.Handle("/app/", http.StripPrefix("/app", http.FileServer(http.Dir(filepathRoot))))

	srvMux.HandleFunc("/healthz", handlerFunc)

	srv := &http.Server{
		Handler: srvMux,
		Addr: ":" + port,
	}

	log.Printf("Serving files from %s on port: %s\n", filepathRoot, port)
	log.Fatal(srv.ListenAndServe())
}

func handlerFunc(w http.ResponseWriter, r *http.Request) {

	// Set Content-Type
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))

}