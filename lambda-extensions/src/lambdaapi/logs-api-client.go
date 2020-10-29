package lambdaapi

import (
	"context"
	"encoding/json"
	"fmt"
)

const (
	// Base URL for extension
	logsURL = "2020-08-15/logs"
	// Subscription Body Constants. Subscribe to platform logs and receive them on ${local_ip}:4243 via HTTP protocol.
	timeoutMs = 1000
	maxBytes  = 262144
	maxItems  = 1000
)

// SubscribeToLogsAPI is - Subscribe to Logs API to receive the Lambda Logs.
func (client *Client) SubscribeToLogsAPI(ctx context.Context, logEvents []string) ([]byte, error) {
	URL := client.baseURL + logsURL

	reqBody, error := json.Marshal(map[string]interface{}{
		"destination": map[string]interface{}{"protocol": "HTTP", "URI": fmt.Sprintf("http://sandbox:%v", ReceiverPort)},
		"types":       logEvents,
		"buffering":   map[string]interface{}{"timeoutMs": timeoutMs, "maxBytes": maxBytes, "maxItems": maxItems},
	})
	if error != nil {
		return nil, error
	}
	headers := map[string]string{
		extensionIdentiferHeader: client.extensionID,
	}
	response, error := client.MakeRequest(ctx, headers, reqBody, "PUT", URL)
	if error != nil {
		return nil, error
	}

	return response, nil
}
