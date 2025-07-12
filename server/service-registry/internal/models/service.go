package models

type Service struct {
	Name    string `json:"name"`
	Address string `json:"address"`
}

type HeartbeatRequest struct {
	Name string `json:"name"`
}

type RegisterRequest struct {
	Name    string `json:"name"`
	Address string `json:"address"`
}

type ErrorResponse struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}

type SuccessResponse struct {
	Status int         `json:"status"`
	Data   interface{} `json:"data"`
} 