package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/dragunovartem99/chessdocs-api/internal/docs"
)

// decodeSubmission reads the request body into a Submission, translating the
// ways that can go wrong into messages a submitter can act on.
func decodeSubmission(w http.ResponseWriter, r *http.Request) (docs.Submission, error) {
	var sub docs.Submission

	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&sub); err != nil {
		return sub, decodeError(err)
	}
	// A second JSON value in the body means the client sent something other
	// than the single object the contract describes.
	if decoder.More() {
		return sub, badRequest("Request body must be a single JSON object")
	}
	return sub, nil
}

func decodeError(err error) error {
	var maxBytes *http.MaxBytesError
	var syntax *json.SyntaxError
	var unmarshal *json.UnmarshalTypeError

	switch {
	case errors.As(err, &maxBytes):
		return &docs.Error{
			Status:  http.StatusRequestEntityTooLarge,
			Message: "Request body is too large",
		}
	case errors.Is(err, io.EOF):
		return badRequest("Request body is empty")
	case errors.As(err, &syntax):
		return badRequest("Request body is not valid JSON")
	case errors.As(err, &unmarshal):
		return badRequest("Field %q has the wrong type", unmarshal.Field)
	default:
		// The decoder reports unknown fields only as a formatted string.
		if field, ok := strings.CutPrefix(err.Error(), "json: unknown field "); ok {
			return badRequest("Unknown field %s", field)
		}
		return badRequest("Request body is not valid JSON")
	}
}

func badRequest(format string, args ...any) error {
	return &docs.Error{Status: http.StatusBadRequest, Message: fmt.Sprintf(format, args...)}
}
