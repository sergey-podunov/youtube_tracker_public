package main

import (
	"log"
	"net/http"
	"youtube_tracker/api"
	yt_http "youtube_tracker/http_handler"
)

func main() {

	httpHandler := &yt_http.StatisticsHttpHandler{}
	srv, err := api.NewServer(httpHandler)
	if err != nil {
		log.Fatal(err)
	}

	if err := http.ListenAndServe(":8080", srv); err != nil {
		log.Fatal(err)
	}
}
