package sumoclient

import (
    "bytes"
    "compress/gzip"
    "errors"
    "net/http"
    "time"
    "fmt"
    // log "github.com/sirupsen/logrus"
    "config"
    // "os"
)

const (
    connectionTimeoutValue  = 10000
    maxRetryAttempts        = 5
    sleepTimeInMilliseconds = 300 * time.Millisecond
)

type SumoLogicClient struct {
    connectionTimeout int
    httpClient        http.Client
    config            *config.LambdaExtensionConfig
}

func NewSumoLogicClient() *SumoLogicClient {
    cfg, _ := config.GetConfig()
    return &SumoLogicClient{
        connectionTimeout: connectionTimeoutValue,
        httpClient:        http.Client{Timeout: time.Duration(connectionTimeoutValue * int(time.Millisecond))},
        config:            cfg,
    }
}

func (s *SumoLogicClient) MakeRequest(buf *bytes.Buffer) (*http.Response, error) {

    request, err := http.NewRequest("POST", s.config.SumoHTTPEndpoint, buf)
    if err != nil {
        fmt.Printf("http.NewRequest() error: %v\n", err)
        return nil, err
    }
    request.Header.Add("Content-Encoding", "gzip")
    request.Header.Add("X-Sumo-Client", "sumologic-lambda-extension")
    // if s.config.sumoName != "" {
    //     request.Header.Add("X-Sumo-Name", s.config.sumoName)
    // }
    // if s.config.sumoHost != "" {
    //     request.Header.Add("X-Sumo-Host", s.config.sumoHost)
    // }
    // if s.config.sumoCategory != "" {
    //     request.Header.Add("X-Sumo-Category", s.config.sumoCategory)
    // }
    response, err := s.httpClient.Do(request)
    return response, err
}
func (s *SumoLogicClient) SendToSumo(logStringToSend string) {
    fmt.Println("Attempting to send to Sumo Endpoint")
    if logStringToSend != "" {
        // compressing
        var buf bytes.Buffer
        g := gzip.NewWriter(&buf)
        g.Write([]byte(logStringToSend))
        g.Close()

        response, err := s.MakeRequest(&buf)

        if (err != nil) || (response.StatusCode != 200 && response.StatusCode != 302 && response.StatusCode < 500) {
            fmt.Println("Not able to post  statuscode:  ", response.StatusCode)
            fmt.Println(fmt.Sprintf("Waiting for %v ms to retry", sleepTimeInMilliseconds))
            time.Sleep(sleepTimeInMilliseconds)

            err := Retry(func(attempt int64) (bool, error) {
                var errRetry error
                response, errRetry = s.MakeRequest(&buf)
                if (errRetry != nil) || (response.StatusCode != 200 && response.StatusCode != 302 && response.StatusCode < 500) {
                    if errRetry == nil {
                        errRetry = errors.New(fmt.Sprintf("Not able to post statuscode: %v", response.StatusCode))
                    }
                    fmt.Println(fmt.Sprintf("Error: %v", errRetry))
                    fmt.Println(fmt.Sprintf("Waiting for %v ms to retry attempts done: %v", sleepTimeInMilliseconds, attempt))
                    time.Sleep(sleepTimeInMilliseconds)
                    return attempt < maxRetryAttempts, errRetry
                } else if response.StatusCode == 200 {
                    fmt.Println(fmt.Sprintf("Post of logs successful after retry %v", attempt))
                    return true, nil
                }
                return attempt < maxRetryAttempts, errRetry
            }, s.config.MaxRetry)
            if err != nil {
                fmt.Println("Finished retrying Error: ", err)
                return
            }
        } else if response.StatusCode == 200 {
            fmt.Println("Post of logs successful")
        }
        if response != nil {
            defer response.Body.Close()
        }
    }
}

//------------------Retry Logic Code-------------------------------

var errMaxRetriesReached = errors.New("exceeded retry limit")

// Func represents functions that can be retried.
type Func func(attempt int64) (retry bool, err error)

// Do keeps trying the function until the second argument
// returns false, or no error is returned.
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

// IsMaxRetries checks whether the error is due to hitting the
// maximum number of retries or not.
func IsMaxRetries(err error) bool {
    return err == errMaxRetriesReached
}

// func main() {
//     os.Setenv("MAX_RETRY", "5")
//     os.Setenv("SUMO_HTTP_ENDPOINT", "https://collectors.sumologic.com/receiver/v1/http/ZaVnC4dhaV2ZZls3q0ihtegxCvl_lvlDNWoNAvTS5BKSjpuXIOGYgu7QZZSd-hkZlub49iL_U0XyIXBJJjnAbl6QK_JX0fYVb_T4KLEUSbvZ6MUArRavYw==")
//     client := NewSumoLogicClient()
//     client.SendToSumo("hello world")

// }
