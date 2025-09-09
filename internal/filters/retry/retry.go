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

package retry

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/vedadiyan/kontrol/internal/pipeline"
)

type RetryFilter struct {
	pipeline.FilterWrapper
	interval time.Duration
	max      int
}

func New(id string, internal time.Duration, max int, opts ...pipeline.FilterOption) *RetryFilter {
	filter := new(RetryFilter)
	filter.FilterId = id

	filter.interval = internal
	filter.max = max

	filter.Handler(filter.do)

	for _, opt := range opts {
		opt(&filter.FilterOptions)
	}

	return filter
}

func (f *RetryFilter) do(ctx context.Context, responseNode pipeline.ResponseNode, request *http.Request) error {
	n := increment(&ctx)
	if n == f.max {
		return f.HandleError(ctx, responseNode, request, fmt.Errorf("failed with maximum retries"))
	}
	<-time.After(f.interval)
	prev := f.PreviousFilter(ctx)
	return prev.Do(ctx, responseNode, request)
}

func increment(ctx *context.Context) int {
	n := (*ctx).Value(pipeline.ContextKey("retry-counter"))
	if n == nil {
		*ctx = context.WithValue(*ctx, pipeline.ContextKey("retry-counter"), 1)
		return 1
	}
	_n := n.(int) + 1
	*ctx = context.WithValue(*ctx, pipeline.ContextKey("retry-counter"), _n)
	return _n
}
