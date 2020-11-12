package utils

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"time"
)

//------------------Retry Logic Code-------------------------------

var errMaxRetriesReached = errors.New("exceeded retry limit")

// Func represents functions that can be retried.
type Func func(attempt int) (retry bool, err error)

// Retry keeps trying the function until the second argument returns false, or no error is returned.
func Retry(fn Func, maxRetries int) error {
	var err error
	var cont bool
	var attempt int = 1
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

// IsTimeRemaining is to check is the lambda is nearing its timeout.
func IsTimeRemaining(deadtime int64) bool {
	t := time.Unix(deadtime, 0)
	dif := time.Now().Sub(t)
	if dif.Seconds() <= 10 {
		return false
	}
	return true
}

// TotalMessagesCountChanged is to check is the lambda function is creating any new logs or not.
func TotalMessagesCountChanged(totalMessages, currentMessages int, duration time.Duration, startTime time.Time) (bool, bool) {
	if totalMessages != 0 && totalMessages == currentMessages {
		t := time.Now().Sub(startTime)
		if t.Milliseconds() >= duration.Milliseconds() {
			return false, true
		}
		return false, false
	}
	return true, false
}
