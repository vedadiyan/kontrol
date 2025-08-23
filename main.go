package main

import (
	"log"
	"net/url"

	"github.com/vedadiyan/kontrol/internal/filters/http"
	"github.com/vedadiyan/kontrol/internal/filters/term"
	"github.com/vedadiyan/kontrol/internal/server"
)

func main() {

	url, err := url.Parse("https://api.restful-api.dev/objects")
	if err != nil {
		log.Fatalln(err)
	}
	filter := http.New(url, nil)
	filter.OnNext(term.New(filter))
	filter.OnFail(term.New(filter))

	server := new(server.Server)
	server.HandleFunc("/test", filter)

	server.ListenAndServe(":8085")
}
