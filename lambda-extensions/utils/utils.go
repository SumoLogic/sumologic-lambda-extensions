package utils

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
)

//------------------Retry Logic Code-------------------------------

var errMaxRetriesReached = errors.New("exceeded retry limit")

// Func represents functions that can be retried.
type Func func(attempt int, prevRetryErr error) (bool, error)

// Retry keeps trying the function until the second argument returns false, or no error is returned.
func Retry(fn Func, maxRetries int, err error) error {

	var cont bool
	var attempt int = 1
	for {
		if attempt > maxRetries {
			return errMaxRetriesReached
		}
		cont, err = fn(attempt, err)
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
func Compress(logStringToSend *string) []byte {

	var buf bytes.Buffer
	g := gzip.NewWriter(&buf)
	g.Write([]byte(*logStringToSend))
	g.Close()
	return buf.Bytes()
}

// CompressBuffer compresses string and returns byte array
func CompressBuffer(inputbuf *bytes.Buffer) *bytes.Buffer {

	var outputbuf bytes.Buffer
	g := gzip.NewWriter(&outputbuf)
	g.Write(inputbuf.Bytes())
	g.Close()
	return &outputbuf
}

// PrettyPrint is to print the object
func PrettyPrint(v interface{}) string {
	data, err := json.MarshalIndent(v, "", "\t")
	if err != nil {
		return ""
	}
	return string(data)
}
