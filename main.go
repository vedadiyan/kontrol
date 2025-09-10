package main

import (
	"log"
	"net/url"
	"time"

	"github.com/vedadiyan/kontrol/internal/filters/http"
	"github.com/vedadiyan/kontrol/internal/filters/inject"
	"github.com/vedadiyan/kontrol/internal/filters/retry"
	"github.com/vedadiyan/kontrol/internal/filters/term"
	"github.com/vedadiyan/kontrol/internal/server"
)

func main() {

	url, err := url.Parse("https://apixxx.restful-api.dev/objects")
	if err != nil {
		log.Fatalln(err)
	}
	filter := http.New("test", url)
	filter.OnNext(http.New("test2", url).OnNext(term.New()))
	filter.OnFail(retry.New("retry", time.Second, 5).OnFail(inject.New("inject").OnNext(term.New())))

	server := new(server.Server)
	server.HandleFunc("/test", filter)

	server.ListenAndServe(":8085")
}
