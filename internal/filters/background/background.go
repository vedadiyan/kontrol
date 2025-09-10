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

package background

import (
	"context"
	"net/http"

	"github.com/vedadiyan/kontrol/internal/pipeline"
)

type options struct {
	pipeline.FilterOption
}

type BackgroundFilter struct {
	pipeline.FilterWrapper
	pipeline.ChainerWrapper
	Filter  pipeline.Filter
	options options
}

type Option func(*options)

func New(id string, wrappedFilter pipeline.Filter, opts ...Option) *BackgroundFilter {
	filter := new(BackgroundFilter)
	filter.FilterId = id
	filter.Handler(filter.Do)
	filter.ChainerWrapper.Filter = filter

	for _, opt := range opts {
		opt(&filter.options)
	}

	return filter
}

func (f *BackgroundFilter) Do(ctx context.Context, responseNode pipeline.ResponseNode, request *http.Request) error {
	clone := responseNode.Clone()
	bgCtx := context.WithoutCancel(ctx)
	go func(ctx context.Context, clone pipeline.ResponseNode) {
		err := f.Filter.Do(ctx, clone, request)
		if err != nil {
			if f, ok := f.Filter.(pipeline.Failer); ok {
				bgCtx = context.WithValue(bgCtx, pipeline.CONTEXT_ERR, err)
				_ = f.Fail().Do(bgCtx, responseNode, request)
				return
			}
			return
		}
		if f, ok := f.Filter.(pipeline.Chainer); ok {
			_ = f.Next().Do(ctx, responseNode, request)
			return
		}
	}(bgCtx, clone)
	return f.HandleNext(ctx, responseNode, request)
}
