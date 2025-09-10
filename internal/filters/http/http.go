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

package http

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/vedadiyan/kontrol/internal/pipeline"
)

type HTTPFilter struct {
	pipeline.FilterWrapper
	pipeline.ChainerWrapper
	pipeline.FailerWrapper
	targetURL *url.URL // The target URL for HTTP requests
}

func New(id string, targetURL *url.URL, opts ...pipeline.FilterOption) *HTTPFilter {
	filter := new(HTTPFilter)
	filter.FilterId = id
	filter.targetURL = targetURL
	filter.Handler(filter.do)
	filter.ChainerWrapper.Filter = filter
	filter.FailerWrapper.Filter = filter

	for _, opt := range opts {
		opt(&filter.FilterOptions)
	}

	return filter
}

func (f *HTTPFilter) do(ctx context.Context, responseNode pipeline.ResponseNode, request *http.Request) error {
	httpResponse, err := performHTTPRequest(ctx, f.targetURL, request)
	if err != nil {
		return f.HandleError(ctx, responseNode, request, err)
	}
	response, err := pipeline.ToResponse(*httpResponse)
	if err != nil {
		return f.HandleError(ctx, responseNode, request, err)
	}

	if !isSuccessStatusCode(response.StatusCode) {
		httpErr := newHTTPError(f.targetURL.String(), httpResponse.StatusCode)
		return f.HandleError(ctx, responseNode.Set(response), request, httpErr)
	}

	return f.HandleNext(ctx, responseNode.Set(response), request)
}

func newHTTPError(url string, statusCode int) error {
	return fmt.Errorf("HTTP request to %s failed with status %d", url, statusCode)
}

func performHTTPRequest(ctx context.Context, targetURL *url.URL, originalRequest *http.Request) (*http.Response, error) {
	clonedRequest := originalRequest.Clone(ctx)
	clonedRequest.URL = targetURL
	clonedRequest.RequestURI = ""

	return http.DefaultClient.Do(clonedRequest)
}

func isSuccessStatusCode(statusCode int) bool {
	return statusCode >= 200 && statusCode < 300
}
