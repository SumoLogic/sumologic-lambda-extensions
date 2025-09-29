package lambdaapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
)

const (
	// Base URL for telemetry api extension
	telemetryURL = "2022-07-01/telemetry"
	// Subscription Body Constants. Subscribe to platform logs and receive them on ${local_ip}:4243 via HTTP protocol.
	//telemetry_receiverPort = 4243
)

// SubscribeToLogsAPI is - Subscribe to Logs API to receive the Lambda Logs.
func (client *Client) SubscribeToTelemetryAPI(ctx context.Context, logEvents []string, telemetryTimeoutMs int, telemetryMaxBytes int64, telemetryMaxItems int) ([]byte, error) {
	URL := client.baseURL + telemetryURL

	reqBody, err := json.Marshal(map[string]interface{}{
		"destination":   map[string]interface{}{"protocol": "HTTP", "URI": fmt.Sprintf("http://sandbox:%v", receiverPort)},
		"types":         logEvents,
		"buffering":     map[string]interface{}{"timeoutMs": telemetryTimeoutMs, "maxBytes": telemetryMaxBytes, "maxItems": telemetryMaxItems},
		"schemaVersion": "2022-07-01",
	})
	if err != nil {
		return nil, err
	}
	headers := map[string]string{
		extensionIdentiferHeader: client.extensionID,
	}
	var response []byte
	if ctx != nil {
		response, err = client.MakeRequestWithContext(ctx, headers, bytes.NewBuffer(reqBody), "PUT", URL)
	} else {
		response, err = client.MakeRequest(headers, bytes.NewBuffer(reqBody), "PUT", URL)
	}
	if err != nil {
		return nil, err
	}

	return response, nil
}
