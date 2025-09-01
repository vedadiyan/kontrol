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
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/vedadiyan/exql"
	"github.com/vedadiyan/exql/lang"
	"github.com/vedadiyan/kontrol/internal/pipeline"
)

type InspectorFilter struct {
	pipeline.FilterWrapper
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
	filter.Handler(filter.Do)

	for _, opt := range opts {
		opt(&filter.FilterOptions)
	}

	return filter
}

func (f *InspectorFilter) Do(ctx context.Context, responseNode pipeline.ResponseNode, request *http.Request) error {
	result, err := f.expr.Evaluate(exql.NewDefaultContext())
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

func createContext(response pipeline.ResponseNode, request *http.Request) exql.DefaultContext {
	ctx := exql.NewDefaultContext()
	rs := exql.NewDefaultContext()
	rs.SetVariable("body", response.Current().Body)
	rs.SetVariable("headers", response.Current().Header)
	rs.SetVariable("statusCode", response.Current().StatusCode)
	rs.SetVariable("trailers", response.Current().Trailer)
	rq := exql.NewDefaultContext()
	body, err := request.GetBody()
	if err != nil {
	}
	data, err := io.ReadAll(body)
	if err != nil {
	}
	rq.SetVariable("body", data)
	rq.SetVariable("headers", request.Header)
	rq.SetVariable("host", request.Host)
	rq.SetVariable("method", request.Method)
	rq.SetVariable("pattern", request.Pattern)
	rq.SetVariable("form", request.PostForm)
	rq.SetVariable("protoMajor", request.ProtoMajor)
	rq.SetVariable("protoMinor", request.ProtoMinor)
	rq.SetVariable("remoteAddr", request.RemoteAddr)
	rq.SetVariable("trailer", request.Trailer)
	rq.SetVariable("transferEncoding", request.TransferEncoding)
	rq.SetVariable("url", request.URL)
	return *ctx
}
