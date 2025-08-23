package http

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/vedadiyan/kontrol/internal/pipeline"
)

type (
	Http struct {
		pipeline.AbstractFilter
		url *url.URL
	}
)

func New(url *url.URL, f pipeline.Filter, opts ...pipeline.FilterOption) *Http {
	out := new(Http)
	out.url = url
	out.Previous = f
	return out
}

func (h *Http) Do(ctx context.Context, rs pipeline.ResponseNode, rq *http.Request) error {
	clone := rq.Clone(ctx)
	clone.URL = h.url
	clone.RequestURI = ""
	res, err := http.DefaultClient.Do(clone)
	if err != nil {
		return errors.Join(err, h.Fail().Do(ctx, rs, rq))
	}
	if res.StatusCode/100 != 2 {
		return errors.Join(fmt.Errorf("request to %s failed with %d", h.url.String(), res.StatusCode), h.Fail().Do(ctx, rs, rq))
	}
	return h.Next().Do(ctx, rs.Set(res), rq)
}
