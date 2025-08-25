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

type BackgroundFilter struct {
	pipeline.AbstractFilter
	Filter pipeline.Filter
}

func New(id string, wrappedFilter pipeline.Filter, previousFilter pipeline.Filter, opts ...pipeline.FilterOption) *BackgroundFilter {
	filter := new(BackgroundFilter)
	filter.FilterId = id
	filter.Filter = wrappedFilter
	filter.Previous = previousFilter

	for _, opt := range opts {
		opt(&filter.FilterOptions)
	}

	return filter
}

func (f *BackgroundFilter) Do(ctx context.Context, responseNode pipeline.ResponseNode, request *http.Request) error {
	go func() {
		clone := responseNode.Clone()
		if err := f.Filter.Do(ctx, clone, request); err != nil {
			_ = f.HandleError(ctx, responseNode, request, err)
		}
	}()
	return f.HandleNext(ctx, responseNode, request)
}
