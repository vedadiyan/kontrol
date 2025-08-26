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
	filter := http.New("test", url)
	filter.OnNext(http.New("test2", url).OnNext(term.New(filter)))
	filter.OnFail(http.New("test2", url).OnNext(term.New(filter)))

	server := new(server.Server)
	server.HandleFunc("/test", filter)

	server.ListenAndServe(":8085")
}
