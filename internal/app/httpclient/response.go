package httpclient

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"

	app "github.com/ach968/x-twitter-cli/internal/app"
)

const maximumUpstreamBodyLength = 4096

var (
	cookieSecretPattern  = regexp.MustCompile(`(?i)(auth_token|ct0|csrf_token)=([^;\s]+)`)
	authorizationPattern = regexp.MustCompile(`(?i)(authorization:\s*)([^\r\n]+)`)
)

func redactJSON(value any) any {
	switch typed := value.(type) {
	case []any:
		for index, child := range typed {
			typed[index] = redactJSON(child)
		}
		return typed
	case map[string]any:
		for key, child := range typed {
			normalized := strings.ReplaceAll(strings.ToLower(key), "-", "_")
			switch normalized {
			case "authorization", "cookie", "auth_token", "ct0", "csrf_token":
				typed[key] = "[REDACTED]"
			default:
				typed[key] = redactJSON(child)
			}
		}
		return typed
	default:
		return value
	}
}

func sanitizeBody(body string) string {
	var value any
	var sanitized string
	if json.Unmarshal([]byte(body), &value) == nil {
		encoded, _ := json.Marshal(redactJSON(value))
		sanitized = string(encoded)
	} else {
		sanitized = cookieSecretPattern.ReplaceAllString(body, "$1=[REDACTED]")
		sanitized = authorizationPattern.ReplaceAllString(sanitized, "$1[REDACTED]")
	}
	if len(sanitized) > maximumUpstreamBodyLength {
		return sanitized[:maximumUpstreamBodyLength]
	}
	return sanitized
}

func resultFromResponse(operation app.OperationName, response app.UpstreamResponse) app.OperationResult {
	sanitized := func() *app.UpstreamResponse {
		copy := response
		copy.Body = sanitizeBody(copy.Body)
		return &copy
	}
	failure := func(code, message string) app.OperationResult {
		return app.OperationResult{OK: false, Error: &app.OperationFailure{Code: code, Message: message}, Upstream: sanitized()}
	}
	switch response.Status {
	case http.StatusUnauthorized:
		return failure("AUTHENTICATION_FAILED", "X rejected the current authentication state")
	case http.StatusTooManyRequests:
		return failure("RATE_LIMITED", "X rate-limited the request")
	}
	if response.Status >= 200 && response.Status < 300 {
		var payload any
		if err := json.Unmarshal([]byte(response.Body), &payload); err != nil {
			return failure("RESPONSE_SHAPE_CHANGED", "X returned a successful response that was not JSON")
		}
		serialized, _ := json.Marshal(payload)
		if strings.Contains(string(serialized), "PersistedQueryNotFound") || strings.Contains(string(serialized), "PERSISTED_QUERY_NOT_FOUND") {
			return app.OperationResult{
				OK: false,
				Error: &app.OperationFailure{
					Code: "CONTRACT_FAILED", Message: "The " + string(operation) + " operation contract was rejected by X",
					RecoveryCommand: "twt contract refresh",
				},
				Upstream: sanitized(),
			}
		}
		return app.OperationResult{OK: true, Payload: payload}
	}
	return failure("UPSTREAM_REJECTED", "X rejected the request, but the cause could not be determined")
}
