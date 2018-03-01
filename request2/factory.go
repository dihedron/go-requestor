// Copyright 2017-present Andrea Funtò. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package request2

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"regexp"
	"strings"

	"github.com/fatih/structs"
)

type operation int8

const (
	add operation = iota
	set
	del
	rem
)

// Factory is the HTTP request factory; it can be used to create child request
// factories, with specialised BaseURLs or other parameters; when a sub-factory
// is generated, it will share the Method and BaseURL by value and all other
// firlds by pointer, so any change in sub-factories will affect the parent too.
type Factory struct {

	// method is the HTTP method to be used for requests generated by this
	// factory.
	method string

	// url is the base URL for generating HTTP requests.
	url string

	// op is used internally to provide a flowing API to header and query parameters
	// maipulation methods.
	op operation

	// headers is a set of header values for HTTP request headers; special headers
	// such as "User-Agent" and "Content-Type" are stored here.
	headers http.Header

	// query is a set of values set in the URL as query parameters.
	parameters url.Values

	// entity is the entity provider; it will be used to generate the request
	// entity as an io.Reader. Moreover, it will be queried to set the request
	// content type.
	body io.Reader
}

// New returns a request factory.
func New(method, url string) *Factory {
	return &Factory{
		method:     method,
		url:        url,
		headers:    map[string][]string{},
		parameters: map[string][]string{},
	}
}

// New clones the current factory and can optionally specify the request method
// and/or the request URL.
func (f *Factory) New(method, url string) *Factory {
	clone := &Factory{
		method:     f.method,
		url:        f.url,
		headers:    f.headers,
		parameters: f.parameters,
		body:       f.body,
	}
	if method != "" {
		clone.method = strings.ToUpper(method)
	}
	if url != "" {
		clone.url = url
	}
	return clone
}

// Base sets the base URL. If you intend to extend the url with Path, the URL
// should be specified with a trailing slash.
func (f *Factory) Base(url string) *Factory {
	f.url = url
	return f
}

// Path overrides the factory URL; absolute and relative URLs can be used.
// TODO: improve documentation showing relative paths
func (f *Factory) Path(path string) *Factory {
	baseURL, baseErr := url.Parse(f.url)
	pathURL, pathErr := url.Parse(path)
	if baseErr == nil && pathErr == nil {
		f.url = baseURL.ResolveReference(pathURL).String()
		return f
	}
	return f
}

// Method sets the default HTTP method for factoory-generated requests.
func (f *Factory) Method(method string) *Factory {
	if method != "" {
		f.method = strings.ToUpper(strings.TrimSpace(method))
	}
	return f
}

// UserAgent sets the user agent information in the request factory; the previous
// value is discarded.
func (f *Factory) UserAgent(userAgent string) *Factory {
	return f.Set().Header("User-Agent", userAgent)
}

// ContentType sets the content type information in the request factory; the
// previous value is discarded.
func (f *Factory) ContentType(contentType string) *Factory {
	return f.Set().Header("Content-Type", contentType)
}

// Add is used to provide a fluent API by which it is possible to add query
// parameters and headers without having many different methods or intermediate
// objects; this method relies on an internal Factory field (named op), which
// will be set to the "add" value and will instruct the following QueryParameter()
// and Header() methods to add the passed values to the current set for the given
// key.
func (f *Factory) Add() *Factory {
	f.op = add
	return f
}

// Set is used to provide a fluent API by which it is possible to replace query
// parameters and headers without having many different methods or intermediate
// objects; this method relies on an internal Factory field (named op), which
// will be set to the "set" value and will instruct the following QueryParameter()
// and Header() methods to replace the current set of values for the given key
// with the passed values.
func (f *Factory) Set() *Factory {
	f.op = set
	return f
}

// Del is used to provide a fluent API by which it is possible to replace query
// parameters and headers without having many different methods or intermediate
// objects; this method relies on an internal Factory field (named op), which
// will be set to the "set" value and will instruct the following QueryParameter()
// and Header() methods to replace the current set of values for the given key
// with the passed values.
func (f *Factory) Del() *Factory {
	f.op = del
	return f
}

// Remove is used to provide a fluent API by which it is possible to remove the
// values of query parameters and headers whose keys match a regular exception.
func (f *Factory) Remove() *Factory {
	f.op = rem
	return f
}

// QueryParameter adds, sets or removes the given set of values to the URL's query
// parameters; if the query parameter is being removed, there is no need to specify
// any value; if the query parameter is being reset, the key is regarded as a
// regular expression.
func (f *Factory) QueryParameter(key string, values ...string) *Factory {
	if f.op == add {
		for _, value := range values {
			f.parameters.Add(key, value)
		}
	} else if f.op == set {
		f.parameters.Del(key)
		for _, value := range values {
			f.parameters.Add(key, value)
		}
	} else if f.op == del {
		f.parameters.Del(key)
	} else if f.op == rem {
		re := regexp.MustCompile(key)
		for key := range f.parameters {
			if re.MatchString(key) {
				defer f.parameters.Del(key)
			}
		}
	}
	return f
}

func (f *Factory) QueryParametersFrom(source interface{}) *Factory {
	switch reflect.ValueOf(source).Kind() {
	case reflect.Struct:
		// do nothing, source is already a struct
	case reflect.Map:
		if m, ok := source.(map[string][]string); ok {
			for key, values := range m {
				for _, value := range values {
					f.parameters.Add(key, value)
				}
			}
		}
	case reflect.Ptr:
		if reflect.ValueOf(source).Elem().Kind() == reflect.Struct {
			source = reflect.ValueOf(source).Elem().Interface()
		} else if reflect.ValueOf(source).Elem().Kind() == reflect.Map {
			source = reflect.ValueOf(source).Elem().Interface()
			if m, ok := source.(map[string][]string); ok {
				for key, values := range m {
					for _, value := range values {
						f.parameters.Add(key, value)
					}
				}
			}
		} else {
			panic("only structs can be passed as sources for query parameters")
		}
	default:
		panic("only structs can be passed as sources for query parameters")
	}

	if p.Tag == "" {
		panic("a valid tag must be provided")
	}

	return scan(p.Tag, source)

	return f
}

// Header adds, sets or removes the given set of values to the URL's headers; if
// the header is being removed, there is no need to specify any value; if the
// header is being reset, the key is regarded as a regular expression.
func (f *Factory) Header(key string, values ...string) *Factory {
	if f.op == add {
		for _, value := range values {
			f.headers.Add(key, value)
		}
	} else if f.op == set {
		f.headers.Del(key)
		for _, value := range values {
			f.headers.Add(key, value)
		}
	} else if f.op == del {
		f.headers.Del(key)
	} else if f.op == rem {
		re := regexp.MustCompile(key)
		for key := range f.headers {
			if re.MatchString(key) {
				defer f.headers.Del(key)
			}
		}
	}
	return f
}

// WithEntity sets the io.Reader from which the request body (payload) will be
// read; if nil is passed, the request will have no payload; the Content-Type
// MUST be provoded separately.
func (f *Factory) WithEntity(entity io.Reader) *Factory {
	f.body = entity
	return f
}

// WithJSONEntity sets an io.Reader that returns a JSON fragment as per the
// input struct; if no Content-Type has been set already, the method will
// automatically set it to "application/json".
func (f *Factory) WithJSONEntity(entity interface{}) io.Reader {

	switch reflect.ValueOf(entity).Kind() {
	case reflect.Struct:
		// do nothing, entity is already a struct, thus it's ok
	case reflect.Ptr:
		// override entity by the value it points to if it's a struct
		if reflect.ValueOf(entity).Elem().Kind() == reflect.Struct {
			entity = reflect.ValueOf(entity).Elem().Interface()
		} else {
			panic("only structs can be passed as source for JSON entities")
		}
	default:
		panic("only structs can be passed as source for JSON entities")
	}

	data, err := json.Marshal(entity)
	if err != nil {
		return nil
	}

	if f.headers.Get("Content-Type") == "" {
		f.ContentType("application/json")
	}

	return bytes.NewReader(data)
}

// WithXMLEntity sets an io.Reader that returns an XML fragment as per the
// input struct; if no Content-Type has been set already, the method will
// automatically set it to "application/xml".
func (f *Factory) WithXMLEntity(entity interface{}) io.Reader {

	switch reflect.ValueOf(entity).Kind() {
	case reflect.Struct:
		// do nothing, entity is already a struct, thus it's ok
	case reflect.Ptr:
		// override entity by the value it points to if it's a struct
		if reflect.ValueOf(entity).Elem().Kind() == reflect.Struct {
			entity = reflect.ValueOf(entity).Elem().Interface()
		} else {
			panic("only structs can be passed as source for XML entities")
		}
	default:
		panic("only structs can be passed as source for XML entities")
	}

	data, err := xml.Marshal(entity)
	if err != nil {
		return nil
	}

	if f.headers.Get("Content-Type") == "" {
		f.ContentType("application/xml")
	}

	return bytes.NewReader(data)
}

// Get sets the factory method to "GET" and returns an http.Request.
func (f *Factory) Get() (*http.Request, error) {
	return f.Method(http.MethodGet).Make()
}

// Post sets the factory method to "POST" and returns an http.Request.
func (f *Factory) Post() (*http.Request, error) {
	return f.Method(http.MethodPost).Make()
}

// Put sets the factory method to "PUT" and returns an http.Request.
func (f *Factory) Put() (*http.Request, error) {
	return f.Method(http.MethodPut).Make()
}

// Patch sets the factory method to "PATCH" and returns an http.Request.
func (f *Factory) Patch() (*http.Request, error) {
	return f.Method(http.MethodPatch).Make()
}

// Delete sets the factory method to "DELETE" and returns an http.Request.
func (f *Factory) Delete() (*http.Request, error) {
	return f.Method(http.MethodDelete).Make()
}

// Head sets the factory method to "HEAD" and returns an http.Request.
func (f *Factory) Head() (*http.Request, error) {
	return f.Method(http.MethodHead).Make()
}

// Trace sets the factory method to "TRACE" and returns an http.Request.
func (f *Factory) Trace() (*http.Request, error) {
	return f.Method(http.MethodTrace).Make()
}

// Options sets the factory method to "OPTIONS" and returns an http.Request.
func (f *Factory) Options() (*http.Request, error) {
	return f.Method(http.MethodOptions).Make()
}

// Connect sets the factory method to "CONNECT" and returns an http.Request.
func (f *Factory) Connect() (*http.Request, error) {
	return f.Method(http.MethodConnect).Make()
}

// Make creates a new http.Request from the information available in the Factory.
func (f *Factory) Make() (*http.Request, error) {

	// parse URL to validate
	url, err := url.Parse(f.url)
	if err != nil {
		return nil, err
	}

	// augment URL with additional query parameters
	url, err = addQueryParameters(url, f.parameters)
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequest(f.method, url.String(), f.body)
	if err != nil {
		return nil, err
	}

	request.Header = f.headers

	return request, nil
}

func addQueryParameters(requestURL *url.URL, parameters url.Values) (*url.URL, error) {
	qp, err := url.ParseQuery(requestURL.RawQuery)
	if err != nil {
		return nil, err
	}
	// encodes query structs into a url.Values map and merges maps
	for key, values := range parameters {
		for _, value := range values {
			qp.Add(key, value)
		}
	}

	// url.Values formats to a sorted "url encoded" string, e.g. "key=val&foo=bar"
	requestURL.RawQuery = qp.Encode()
	return requestURL, nil
}

// scan is the actual workhorse method: it scans the source struct for tagged
// headers and extracts their values; if any embedded or child struct is
// encountered, it is scanned for values.
func scan(key string, source interface{}) map[string][]interface{} {
	result := map[string][]interface{}{}
	for _, field := range structs.Fields(source) {
		if field.IsEmbedded() || field.Kind() == reflect.Struct {
			for k, v := range scan(key, field.Value()) {
				if values, ok := result[k]; ok {
					result[k] = append(values, v...)
				} else {
					result[k] = v
				}
			}
		} else {
			tag := field.Tag(key)
			if tag != "" {
				value := field.Value()
				if values, ok := result[tag]; ok {
					result[tag] = append(values, value)
				} else {
					result[tag] = []interface{}{value}
				}
			}
		}
	}
	return result
}
