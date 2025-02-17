package http_handler

import (
	"context"
	"youtube_tracker/api"
)

type BooksService struct {
	api.UnimplementedHandler
}

func (s *BooksService) BooksIDGet(ctx context.Context, params api.BooksIDGetParams) (api.BooksIDGetRes, error) {
	return &api.Book{
		Author: "<NAME>",
		Title:  "The Hobbit",
	}, nil
}
