package internal

import (
	"net/http"
)

type (
	Subscriber func(w *http.Response, r *http.Request) bool
)

var (
	_subscribers map[string][]Subscriber
)

func init() {
	_subscribers = make(map[string][]Subscriber)
}

func ListenAndServe(addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", handler)
	return http.ListenAndServe(addr, mux)
}

func handler(w http.ResponseWriter, r *http.Request) {
	res := new(http.Response)
	for _, subscriber := range _subscribers[r.Pattern] {
		if !subscriber(res, r) {
			res.Write(w)
			return
		}
		body, err := r.GetBody()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		r.Body = body
	}
}

func Subscribe(pattern string, subscriber Subscriber) {
	_subscribers[pattern] = append(_subscribers[pattern], subscriber)
}
