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