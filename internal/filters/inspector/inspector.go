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

package inspector

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/vedadiyan/exql"
	"github.com/vedadiyan/exql/lang"
	_http "github.com/vedadiyan/exql/lib/http"
	"github.com/vedadiyan/kontrol/internal/pipeline"
)

type InspectorFilter struct {
	pipeline.FilterWrapper
	pipeline.ChainerWrapper
	pipeline.FailerWrapper
	expr lang.ExprNode
}

func New(id string, query string, opts ...pipeline.FilterOption) *InspectorFilter {
	filter := new(InspectorFilter)
	filter.FilterId = id
	expr, err := exql.Parse(query)
	if err != nil {
		panic(err)
	}
	filter.expr = expr
	filter.Handler(filter.do)
	filter.ChainerWrapper.Filter = filter
	filter.FailerWrapper.Filter = filter

	for _, opt := range opts {
		opt(&filter.FilterOptions)
	}

	return filter
}

func (f *InspectorFilter) do(ctx context.Context, responseNode pipeline.ResponseNode, request *http.Request) error {
	result, err := f.expr.Evaluate(createContext(responseNode, request))
	if err != nil {
		return f.HandleError(ctx, responseNode, request, err)
	}
	boolean, ok := result.(bool)
	if !ok {
		return f.HandleError(ctx, responseNode, request, fmt.Errorf("expected boolean but got %T", result))
	}
	if !boolean {
		return f.HandleError(ctx, responseNode, request, fmt.Errorf("inspection did not meet expectation"))
	}
	return f.HandleNext(ctx, responseNode, request)
}

func createContext(response pipeline.ResponseNode, request *http.Request) lang.Context {
	ctx := exql.NewDefaultContext()
	currentResponse := response.Current()
	res := new(http.Response)
	res.Body = io.NopCloser(bytes.NewBuffer(currentResponse.Body))
	res.StatusCode = currentResponse.StatusCode
	res.Header = currentResponse.Header
	res.Trailer = currentResponse.Trailer
	ctx.SetVariable("request", _http.New(request))
	ctx.SetVariable("response", _http.New(res))
	return ctx
}
