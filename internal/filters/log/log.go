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

package log

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/vedadiyan/exql"
	"github.com/vedadiyan/exql/lang"
	_http "github.com/vedadiyan/exql/lib/http"
	"github.com/vedadiyan/kontrol/internal/pipeline"
)

type LogFilter struct {
	pipeline.FilterWrapper
	expr lang.ExprNode
}

func New(id string, query string, opts ...pipeline.FilterOption) *LogFilter {
	filter := new(LogFilter)
	filter.FilterId = id
	expr, err := exql.Parse(query)
	if err != nil {
		panic(err)
	}
	filter.expr = expr
	filter.Handler(filter.do)

	for _, opt := range opts {
		opt(&filter.FilterOptions)
	}

	return filter
}

func (f *LogFilter) do(ctx context.Context, responseNode pipeline.ResponseNode, request *http.Request) error {
	result, err := f.expr.Evaluate(createContext(ctx, responseNode, request))
	switch err {
	case nil:
		{
			log.Println(result)
		}
	default:
		{
			log.Println(err)
		}
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
