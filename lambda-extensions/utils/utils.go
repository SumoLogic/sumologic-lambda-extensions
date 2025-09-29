package utils

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
)

//------------------Retry Logic Code-------------------------------

var errMaxRetriesReached = errors.New("exceeded retry limit")

// Func represents functions that can be retried.
type Func func(attempt int) (retry bool, err error)

// Retry keeps trying the function until the second argument returns false, or no error is returned.
func Retry(fn Func, maxRetries int) error {
	var err error
	var cont bool
	var attempt = 1
	for {
		if attempt > maxRetries {
			return errMaxRetriesReached
		}
		cont, err = fn(attempt)
		if !cont || err == nil {
			break
		}
		attempt++

	}
	return err
}

// StringInSlice checks string present in slice of strings
func StringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

// Compress compresses string and returns byte array
func Compress(logStringToSend *string) ([]byte, error) {
	var buf bytes.Buffer
	g := gzip.NewWriter(&buf)

	if _, err := g.Write([]byte(*logStringToSend)); err != nil {
		return nil, fmt.Errorf("failed to write log string: %w", err)
	}

	if err := g.Close(); err != nil {
		return nil, fmt.Errorf("failed to close gzip writer: %w", err)
	}

	return buf.Bytes(), nil
}

// CompressBuffer compresses string and returns byte array
func CompressBuffer(inputbuf *bytes.Buffer) (*bytes.Buffer, error) {
	var outputbuf bytes.Buffer
	g := gzip.NewWriter(&outputbuf)

	if _, err := g.Write(inputbuf.Bytes()); err != nil {
		return nil, fmt.Errorf("failed to compress buffer: %w", err)
	}

	if err := g.Close(); err != nil {
		return nil, fmt.Errorf("failed to close gzip writer: %w", err)
	}

	return &outputbuf, nil
}

// PrettyPrint is to print the object
func PrettyPrint(v interface{}) string {
	data, err := json.MarshalIndent(v, "", "\t")
	if err != nil {
		return ""
	}
	return string(data)
}

// ParseJson to determine whether a string is valid JSON
func ParseJson(s string) (js map[string]interface{}, err error) {
	err = json.Unmarshal([]byte(s), &js)
	return
}
