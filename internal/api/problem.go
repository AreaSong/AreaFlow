package api

import (
	"fmt"
	"net/http"
)

type problemDetails struct {
	Type      string `json:"type"`
	Title     string `json:"title"`
	Status    int    `json:"status"`
	Detail    string `json:"detail"`
	Error     string `json:"error"`
	RequestID string `json:"request_id,omitempty"`
}

func problemFor(status int, detail, requestID string) problemDetails {
	title := http.StatusText(status)
	if title == "" {
		title = "Request failed"
	}
	return problemDetails{
		Type: "https://areasong.dev/problems/" + fmt.Sprint(status), Title: title,
		Status: status, Detail: detail, Error: detail, RequestID: requestID,
	}
}
