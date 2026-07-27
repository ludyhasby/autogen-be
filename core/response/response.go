package coreresponse

import "time"

type ApiResponse[T any] struct {
	Latency string    `json:"latency"`
	Error   *string   `json:"error"`
	Tin     time.Time `json:"tin"`
	Tout    time.Time `json:"tout"`
	Data    *T        `json:"data"`
	Success bool      `json:"success"`
	Status  int       `json:"status"`
}

func (r *ApiResponse[T]) GetStatus() int {
	if r.Status == 0 {
		return 200
	}
	return r.Status
}

type ErrorResponse struct {
	Status  int `json:"status"`
	Message any `json:"message"`
	Error   any `json:"error"`
}

func (r *ErrorResponse) GetStatusCode() int {
	return r.Status
}
