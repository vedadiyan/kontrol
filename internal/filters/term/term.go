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

func New(f pipeline.Filter) *Term {
	out := new(Term)
	out.Handler(out.Do)
	return out
}

func (h *Term) Do(ctx context.Context, rs pipeline.ResponseNode, rq *http.Request) error {
	return nil
}
