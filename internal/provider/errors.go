package provider

import (
	"encoding/json"
	"fmt"
	"strings"
)

// parseProviderError extracts a human-readable error from provider API responses.
func parseProviderError(providerName string, statusCode int, body []byte) string {
	// Try to parse JSON error
	var errResp struct {
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
			Code    string `json:"code"`
		} `json:"error"`
		Message string `json:"message"`
	}
	if json.Unmarshal(body, &errResp) == nil {
		msg := errResp.Error.Message
		if msg == "" {
			msg = errResp.Message
		}
		if msg != "" {
			return msg
		}
	}

	// Friendly messages for common status codes
	switch statusCode {
	case 401:
		return "authentication failed — check your API key"
	case 403:
		return "access denied — your API key may not have the required permissions"
	case 404:
		return "model or endpoint not found"
	case 429:
		return "rate limited — too many requests, please wait"
	case 500:
		return "internal server error on the provider side"
	case 502, 503:
		return "provider service temporarily unavailable"
	case 529:
		return "provider is overloaded, please try again later"
	}

	// Fallback
	s := string(body)
	if len(s) > 200 {
		s = s[:200] + "..."
	}
	return fmt.Sprintf("HTTP %d: %s", statusCode, s)
}

// friendlyProviderError converts common network errors to user-friendly messages.
func friendlyProviderError(err error) string {
	msg := err.Error()
	if strings.Contains(msg, "connection refused") {
		return "connection refused (is the service running?)"
	}
	if strings.Contains(msg, "no such host") {
		return "host not found (check the URL)"
	}
	if strings.Contains(msg, "timeout") || strings.Contains(msg, "deadline exceeded") {
		return "connection timed out (service may be starting up)"
	}
	if strings.Contains(msg, "EOF") {
		return "connection closed unexpectedly"
	}
	if strings.Contains(msg, "reset by peer") {
		return "connection reset by server"
	}
	return msg
}
