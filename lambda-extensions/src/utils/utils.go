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
type Func func(attempt int64) (retry bool, err error)

// Retry keeps trying the function until the second argument returns false, or no error is returned.
func Retry(fn Func, maxRetries int64) error {
	var err error
	var cont bool
	var attempt int64 = 1
	for {
		cont, err = fn(attempt)
		if !cont || err == nil {
			break
		}
		attempt++
		if attempt > maxRetries {
			return errMaxRetriesReached
		}
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

// Compress compresses string and returns buffer
func Compress(logStringToSend *string) []byte {

	var buf bytes.Buffer
	g := gzip.NewWriter(&buf)
	g.Write([]byte(*logStringToSend))
	g.Close()
	return buf.Bytes()
}

// PrettyPrint is to print the object
func PrettyPrint(v interface{}) string {
	data, err := json.MarshalIndent(v, "", "\t")
	if err != nil {
		return ""
	}
	return string(data)
}
