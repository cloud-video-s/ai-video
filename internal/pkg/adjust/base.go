// Package adjust provides a client for the Adjust Campaign API.
package adjust

//func readResponse(reader io.Reader, limit int64) ([]byte, error) {
//	body, err := io.ReadAll(io.LimitReader(reader, limit+1))
//	if err != nil {
//		return nil, fmt.Errorf("read adjust response: %w", err)
//	}
//	if int64(len(body)) > limit {
//		return nil, fmt.Errorf("adjust response exceeds %d bytes", limit)
//	}
//	return body, nil
//}
//
//func parseAPIError(statusCode int, body []byte) error {
//	var envelope struct {
//		Error            json.RawMessage `json:"error"`
//		Code             any             `json:"code"`
//		ErrorCode        any             `json:"error_code"`
//		Message          string          `json:"message"`
//		ErrorDescription string          `json:"error_description"`
//		ErrorDesc        string          `json:"error_desc"`
//		RequestID        string          `json:"request_id"`
//	}
//	_ = json.Unmarshal(body, &envelope)
//
//	code := envelope.Code
//	if code == nil {
//		code = envelope.ErrorCode
//	}
//	message := firstNonEmpty(envelope.Message, envelope.ErrorDescription, envelope.ErrorDesc)
//	if len(envelope.Error) > 0 && string(envelope.Error) != "null" {
//		var detail struct {
//			Code      any    `json:"code"`
//			Message   string `json:"message"`
//			RequestID string `json:"request_id"`
//		}
//		if err := json.Unmarshal(envelope.Error, &detail); err == nil {
//			if code == nil {
//				code = detail.Code
//			}
//			message = firstNonEmpty(detail.Message, message)
//			envelope.RequestID = firstNonEmpty(envelope.RequestID, detail.RequestID)
//		} else {
//			var text string
//			if json.Unmarshal(envelope.Error, &text) == nil {
//				message = firstNonEmpty(text, message)
//			}
//		}
//	}
//	if message == "" {
//		message = strings.TrimSpace(string(body))
//		if len(message) > 500 {
//			message = message[:500]
//		}
//	}
//
//	return &APIError{
//		StatusCode: statusCode,
//		Code:       stringify(code),
//		Message:    message,
//		RequestID:  strings.TrimSpace(envelope.RequestID),
//	}
//}
//
