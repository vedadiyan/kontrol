package term

import (
	"context"
	"net/http"

	"github.com/vedadiyan/kontrol/internal/pipeline"
)

type (
	Term struct {
		pipeline.FilterWrapper
	}
)

func New() *Term {
	out := new(Term)
	out.Handler(out.do)
	return out
}

func (h *Term) do(ctx context.Context, rs pipeline.ResponseNode, rq *http.Request) error {
	return nil
}
