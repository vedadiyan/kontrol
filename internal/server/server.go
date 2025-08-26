package server

import (
	"context"
	"log"
	"net/http"

	"github.com/vedadiyan/kontrol/internal/pipeline"
)

type (
	Server struct {
		mux http.ServeMux
	}
)

func (server *Server) ListenAndServe(addr string) {
	http.ListenAndServe(addr, &server.mux)
}

func (server *Server) HandleFunc(pattern string, logic pipeline.Filter) {
	server.mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		response := pipeline.DefaultResponseNode(new(pipeline.Response))
		if err := logic.Do(context.TODO(), response, r); err != nil {
			log.Println(err)
		}

		for {
			next := response.Next()
			if next == nil {
				break
			}
			if next.Current() != nil {
				break
			}
			response = next
		}

		response.Current().Header.Write(w)
		response.Current().Trailer.Write(w)
		w.Write(response.Current().Body)
	})
}
