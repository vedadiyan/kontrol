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

func (server *Server) HandleFunc(pattern string, logic pipeline.Filter) {
	server.mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		httpResponse := new(http.Response)
		httpResponse.Proto = r.Proto
		httpResponse.ProtoMajor = r.ProtoMajor
		httpResponse.ProtoMinor = r.ProtoMinor
		httpResponse.Header = make(http.Header)
		response := pipeline.DefaultResponseNode(httpResponse)
		if err := logic.Do(context.TODO(), response, r); err != nil {
			log.Println(err)
		}
		for response.Next() != nil {
			response = response.Next()
		}
		response.Current().Write(w)
	})
}
