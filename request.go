package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
)

// Request represents a Skylark API request
type Request struct {
	Endpoint         string
	Collection       string
	ID               string
	Fields           map[string]*Field
	ctx              context.Context
	additionalFields map[string]string
}

// NewRequest returns a simple request with the given
func NewRequest(endpoint, collection, id string) *Request {
	return &Request{
		Collection: collection, Endpoint: endpoint, Fields: make(map[string]*Field), additionalFields: make(map[string]string), ID: id, ctx: context.Background()}
}

// AddField adds a field to the request.
// If a request has fields specified it will only return those fields.
func (r *Request) AddField(f *Field) *Request {
	r.Fields[f.Name] = f
	return r
}

// QueryParams calculates and returns the request's query parameters.
func (r *Request) QueryParams() url.Values {
	v := url.Values{}
	for _, field := range r.Fields {
		v = field.apply(v)
	}
	for key, value := range r.additionalFields {
		v.Add(key, value)
	}
	return v
}

// ToURL converts the request into a url.URL
func (r *Request) ToURL() (*url.URL, error) {
	temp := r.Endpoint + r.Collection + "/"
	if r.ID != "" {
		temp += r.ID + "/"
	}
	queryParams := r.QueryParams().Encode()
	if queryParams != "" {
		temp += "?" + queryParams
	}
	return url.Parse(temp)
}

// WithContext set's the context the request will be executed with.
// Panics on nil context
func (r *Request) WithContext(ctx context.Context) *Request {
	if ctx == nil {
		panic("nil context")
	}
	r.ctx = ctx
	return r
}

// OrderBy sorts the response by the given field
func (r *Request) OrderBy(f *Field) *Request {
	r.additionalFields["order"] = f.Name
	return r
}

// WithFilter allows to filter by a field that is not in the requested response
func (r *Request) WithFilter(fieldName string, filter *Filter) *Request {
	if filter.c != "" {
		fieldName = fmt.Sprintf("%s__%s", fieldName, filter.c)
	}
	r.additionalFields[fieldName] = filter.value
	return r
}

// Expand expands a field without explicitly listing it as a field to return.
// This is usefult if you want to return all fields.
func (r *Request) Expand(f *Field) *Request {
	f.IsExpanded = true
	f.IsIncluded = false
	r.AddField(f)
	return r
}

// Execute executes the request and writes it's results to the value pointed to by v.
func (r *Request) Execute(v interface{}) error {
	url, err := r.ToURL()
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(r.ctx, "GET", url.String(), nil)
	if err != nil {
		return err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		message, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return fmt.Errorf("Unable to read error message from server: %w", err)
		}
		return errors.New(string(message))
	}

	return json.NewDecoder(res.Body).Decode(v)
}
