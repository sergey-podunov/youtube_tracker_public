package main

import (
	"log"
	"net/http"
	"youtube_tracker/api"
	yt_http "youtube_tracker/http_handler"
)

func main() {

	// Create http_handler instance.
	bookService := &yt_http.BooksService{}
	// Create generated server.
	srv, err := api.NewServer(bookService)
	if err != nil {
		log.Fatal(err)
	}

	if err := http.ListenAndServe(":8080", srv); err != nil {
		log.Fatal(err)
	}
}
