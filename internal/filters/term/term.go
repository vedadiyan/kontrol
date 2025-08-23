package term

import (
	"context"
	"net/http"

	"github.com/vedadiyan/kontrol/internal/pipeline"
)

type (
	Term struct {
		pipeline.AbstractFilter
	}
)

func New(f pipeline.Filter) *Term {
	out := new(Term)
	out.Previous = f
	return out
}

func (h *Term) Do(ctx context.Context, rs pipeline.ResponseNode, rq *http.Request) error {
	return nil
}
