package main

import (
	"log"
	"net/http"
	"youtube_tracker/internal/api"
	yt_http "youtube_tracker/internal/http_handler"
)

func main() {

	httpHandler := &yt_http.MainHttpHandler{}
	srv, err := api.NewServer(httpHandler)
	if err != nil {
		log.Fatal(err)
	}

	if err := http.ListenAndServe(":8080", srv); err != nil {
		log.Fatal(err)
	}
}
