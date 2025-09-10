// Copyright (c) 2025 Pouya Vedadiyan. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cache

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/vedadiyan/exql"
	"github.com/vedadiyan/exql/lang"
	_http "github.com/vedadiyan/exql/lib/http"
	"github.com/vedadiyan/kontrol/internal/pipeline"
)

type CacheType int

type Cache interface {
	Get(key string) (*pipeline.Response, error)
	Set(key string, value *pipeline.Response, ttl time.Duration) error
}

type options struct {
	pipeline.FilterOption
	ttl time.Duration
}

type GetCacheFilter struct {
	pipeline.FilterWrapper
	pipeline.ChainerWrapper
	pipeline.FailerWrapper
	expr    lang.ExprNode
	cache   Cache
	options options
}

type Option func(*options)

const (
	CACHE_GET CacheType = iota
	CACHE_SET
)

func New(id string, cacheType CacheType, key string, cache Cache, ttl time.Duration, opts ...Option) *GetCacheFilter {
	filter := new(GetCacheFilter)
	filter.FilterId = id
	expr, err := exql.Parse(key)
	if err != nil {
		panic(err)
	}
	filter.expr = expr
	filter.cache = cache
	switch cacheType {
	case CACHE_GET:
		{
			filter.Handler(filter.get)
		}
	case CACHE_SET:
		{
			filter.Handler(filter.set)
		}
	default:
		{
			panic("unexpected cache type value")
		}
	}
	filter.ChainerWrapper.Filter = filter
	filter.FailerWrapper.Filter = filter

	for _, opt := range opts {
		opt(&filter.options)
	}

	return filter
}

func (f *GetCacheFilter) get(ctx context.Context, responseNode pipeline.ResponseNode, request *http.Request) error {
	result, err := f.expr.Evaluate(createContext(ctx, responseNode, request))
	if err != nil {
		return f.HandleError(ctx, responseNode, request, err)
	}
	value, err := f.cache.Get(fmt.Sprintf("%v", result))
	if err != nil {
		return f.HandleError(ctx, responseNode, request, err)
	}
	return f.HandleNext(ctx, responseNode.Set(value), request)
}

func (f *GetCacheFilter) set(ctx context.Context, responseNode pipeline.ResponseNode, request *http.Request) error {
	result, err := f.expr.Evaluate(createContext(ctx, responseNode, request))
	if err != nil {
		return f.HandleError(ctx, responseNode, request, err)
	}
	err = f.cache.Set(fmt.Sprintf("%v", result), responseNode.Current(), f.options.ttl)
	if err != nil {
		return f.HandleError(ctx, responseNode, request, err)
	}
	return f.HandleNext(ctx, responseNode, request)
}

func createContext(ctx context.Context, response pipeline.ResponseNode, request *http.Request) lang.Context {
	_ctx := exql.NewDefaultContext(exql.WithBuiltInLibrary(), exql.WithFunctions(
		map[string]lang.Function{
			"contextKey": func(args []lang.Value) (lang.Value, error) {
				return lang.StringValue(fmt.Sprintf("%v", ctx.Value(pipeline.ContextKey(fmt.Sprintf("%v", args[0]))))), nil
			},
		},
	))
	currentResponse := response.Current()
	res := new(http.Response)
	res.Body = io.NopCloser(bytes.NewBuffer(currentResponse.Body))
	res.StatusCode = currentResponse.StatusCode
	res.Header = currentResponse.Header
	res.Trailer = currentResponse.Trailer
	_ctx.SetVariable("request", _http.New(request))
	_ctx.SetVariable("response", _http.New(res))
	return _ctx
}
