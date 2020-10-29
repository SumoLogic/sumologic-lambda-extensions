package lambdaapi

import (
	"fmt"
	"io/ioutil"
	"net/http"
)

var (
	Messages = make(chan string, 10)
)

const (
	// ReceiverIP is Web Server Constants
	ReceiverIP = "0.0.0.0"
	// ReceiverPort is Web Server Constants
	ReceiverPort = 4243
)

// HTTPServer is a struct with list
type HTTPServer struct {
<<<<<<< HEAD
}

// NewHTTPServer is to return a new object
func NewHTTPServer() *HTTPServer {
	return &HTTPServer{}
=======
	dataQueue chan []byte
}

// NewHTTPServer is to return a new object
func NewHTTPServer(consumerQueue chan []byte) *HTTPServer {
	return &HTTPServer{dataQueue: consumerQueue}
>>>>>>> e6da5b0bd18066da44e9d174e11549331ae902a0
}

// HTTPServerStart is to start the HTTP Server
func (httpServer *HTTPServer) HTTPServerStart() {
	http.HandleFunc("/", httpServer.LogsHandler)
	http.ListenAndServe(fmt.Sprintf("%s:%d", ReceiverIP, ReceiverPort), nil)
}

// LogsHandler is Server Implementation to get Logs from logs API.
func (httpServer *HTTPServer) LogsHandler(writer http.ResponseWriter, request *http.Request) {
	if request.URL.Path != "/" {
		http.NotFound(writer, request)
		return
	}
	switch request.Method {
	case "POST":
<<<<<<< HEAD
		reqBody, err := ioutil.ReadAll(request.Body)
		if err != nil {
			log.Fatalln(err)
		}
		Messages <- string(reqBody)
=======
		reqBody, error := ioutil.ReadAll(request.Body)
		if error != nil {
			panic(error)
		}
		fmt.Println("Producing data into dataQueue")
		httpServer.dataQueue <- []byte(reqBody)
>>>>>>> e6da5b0bd18066da44e9d174e11549331ae902a0
	}
}
