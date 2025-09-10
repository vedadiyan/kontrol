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

package inject

import (
	"context"
	"net/http"

	"github.com/vedadiyan/kontrol/internal/pipeline"
)

type InjectFilter struct {
	pipeline.FilterWrapper
	pipeline.ChainerWrapper
	body       []byte
	statusCode int
	headers    http.Header
	trailers   http.Header
}

func New(id string, body []byte, statusCode int, headers http.Header, trailers http.Header, opts ...pipeline.FilterOption) *InjectFilter {
	filter := new(InjectFilter)
	filter.FilterId = id

	filter.Handler(filter.do)
	filter.ChainerWrapper.Filter = filter

	filter.body = body
	filter.statusCode = statusCode
	filter.headers = headers
	filter.trailers = trailers

	for _, opt := range opts {
		opt(&filter.FilterOptions)
	}

	return filter
}

func (f *InjectFilter) do(ctx context.Context, responseNode pipeline.ResponseNode, request *http.Request) error {
	current := responseNode.Current()
	if f.body != nil {
		current.Body = f.body
	}
	if f.statusCode != 0 {
		current.StatusCode = f.statusCode
	}
	if f.headers != nil {
		current.Header = f.headers
	}
	if f.trailers != nil {
		current.Trailer = f.trailers
	}
	return f.HandleNext(ctx, responseNode, request)
}
