package apiserv

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/missionMeteora/toolkit/errors"
)

// Common responses
var (
	RespNotFound   = NewErrorResponse(http.StatusNotFound, http.StatusText(http.StatusNotFound))
	RespForbidden  = NewErrorResponse(http.StatusForbidden, http.StatusText(http.StatusForbidden))
	RespBadRequest = NewErrorResponse(http.StatusBadRequest, http.StatusText(http.StatusBadRequest))
)

// Common mime-types
const (
	MimeJSON  = "application/json; charset=utf-8"
	MimeHTML  = "text/html; charset=utf-8"
	MimePlain = "text/plain; charset=utf-8"
)

// NewResponse returns a new success response (code 200) with the specific data
func NewResponse(data interface{}) *Response {
	return &Response{
		Code: 200,
		Data: data,
	}
}

// ReadResponse reads a response from an io.ReadCloser and closes the body.
func ReadResponse(rc io.ReadCloser) (*Response, error) {
	var r Response
	if err := json.NewDecoder(rc).Decode(&r); err != nil {
		rc.Close()
		return nil, err
	}
	rc.Close()
	return &r, nil
}

// Response is the default standard api response
type Response struct {
	Code    int         `json:"code"`    // if code is not set, it defaults to 200 if error is nil otherwise 400.
	Success bool        `json:"success"` // automatically set to true if r.Code >= 200 && r.Code < 300.
	Data    interface{} `json:"data,omitempty"`
	Errors  []*Error    `json:"errors,omitempty"`

	Indent bool `json:"-"` // if set to true, the json encoder will output indented json.
}

// ErrorList returns an errors.ErrorList of this response's errors or nil.
func (r *Response) ErrorList() *errors.ErrorList {
	if len(r.Errors) == 0 {
		return nil
	}
	var el errors.ErrorList
	for _, err := range r.Errors {
		el.Push(err)
	}
	return &el
}

// WriteToCtx writes the response to a ResponseWriter
func (r *Response) WriteToCtx(ctx *Context) error {
	if r.Code == 0 {
		if len(r.Errors) > 0 {
			r.Code = http.StatusBadRequest
		} else {
			r.Code = http.StatusOK
		}
	}

	r.Success = r.Code >= http.StatusOK && r.Code < http.StatusMultipleChoices

	return ctx.JSON(r.Code, r.Indent, r)
}

// NewErrorResponse returns a new error response.
// errs can be:
// 1. string or []byte
// 2. error
// 3. Error / *Error
// 4. any other value will be used as-is
func NewErrorResponse(code int, errs ...interface{}) *Response {
	resp := &Response{
		Code:   code,
		Errors: make([]*Error, len(errs)),
	}

	for i, err := range errs {
		switch v := err.(type) {
		case Error:
			resp.Errors[i] = &v
		case *Error:
			resp.Errors[i] = v
		case string:
			resp.Errors[i] = &Error{Message: v}
		case []byte:
			resp.Errors[i] = &Error{Message: string(v)}
		case error:
			resp.Errors[i] = &Error{Message: v.Error()}
		default:
			log.Panicf("unsupported error type (%T): %v", v, v)
		}
	}

	return resp
}

// Error is returned in the error field of a Response.
type Error struct {
	Message   string `json:"message,omitempty"`
	Field     string `json:"field,omitempty"`
	IsMissing bool   `json:"isMissing,omitempty"`
}

func (e *Error) Error() string {
	if e.Message != "" {
		return e.Message
	}

	if e.Field != "" && e.IsMissing {
		return `field "` + e.Field + `" is missing`
	}

	return fmt.Sprintf("Error{Message: %q, Field: %q, IsMissing: %v}", e.Message, e.Field, e.IsMissing)
}
