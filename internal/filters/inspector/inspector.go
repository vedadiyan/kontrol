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
	"net/http"

	eval "github.com/vedadiyan/exql/lang"
	"github.com/vedadiyan/kontrol/internal/pipeline"
)

type InspectorFilter struct {
	pipeline.AbstractFilter
	Query string
}

type Context struct {
}

func New(id string, query string, previousFilter pipeline.Filter, opts ...pipeline.FilterOption) *InspectorFilter {
	filter := new(InspectorFilter)
	filter.FilterId = id
	filter.Query = query
	filter.Previous = previousFilter

	for _, opt := range opts {
		opt(&filter.FilterOptions)
	}

	return filter
}

func (f *InspectorFilter) Do(ctx context.Context, responseNode pipeline.ResponseNode, request *http.Request) error {
	expr, err := eval.ParseExpression(f.Query)
	if err != nil {
		return f.HandleError(ctx, responseNode, request, err)
	}

	context := new(Context)
	result := expr.Evaluate(context)
	boolean, ok := result.(bool)
	if !ok {
		return f.HandleError(ctx, responseNode, request, fmt.Errorf("expected boolean but got %T", result))
	}
	if !boolean {
		return f.HandleError(ctx, responseNode, request, fmt.Errorf("inspection did not meet expectation"))
	}
	return f.HandleNext(ctx, responseNode, request)
}

func (c *Context) GetVariable(name string) eval.Value {
	return nil
}
func (c *Context) GetFunction(name string) eval.Function {
	return nil
}
