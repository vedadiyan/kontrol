package http

import (
	"context"
	"errors"
	"net/http"

	"github.com/vedadiyan/kontrol/internal/pipeline"
)

type (
	Http struct {
		pipeline.AbstractFilter
	}
)

func New(f pipeline.Filter) *Http {
	out := new(Http)
	out.Previous = f
	return out
}

func (h *Http) Do(ctx context.Context, rs pipeline.ResponseNode, rq *http.Request) error {
	clone := rq.Clone(ctx)
	res, err := http.DefaultClient.Do(clone)
	if err != nil {
		return errors.Join(err, h.Fail().Do(ctx, rs, rq))
	}
	return h.Next().Do(ctx, rs.Set(res), rq)
}
