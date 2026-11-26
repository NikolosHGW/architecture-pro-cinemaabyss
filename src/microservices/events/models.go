package main

import (
	"encoding/json"
	"time"
)

type movieEvent struct {
	MovieID     *int64     `json:"movie_id"`
	Title       *string    `json:"title"`
	Action      *string    `json:"action"`
	UserID      *int64     `json:"user_id,omitempty"`
	Rating      *float64   `json:"rating,omitempty"`
	Genres      *[]*string `json:"genres,omitempty"`
	Description *string    `json:"description,omitempty"`
}

type userEvent struct {
	UserID    *int64     `json:"user_id"`
	Action    *string    `json:"action"`
	Timestamp *time.Time `json:"timestamp"`
	Username  *string    `json:"username,omitempty"`
	Email     *string    `json:"email,omitempty"`
}

type paymentEvent struct {
	PaymentID  *int64     `json:"payment_id"`
	UserID     *int64     `json:"user_id"`
	Amount     *float64   `json:"amount"`
	Status     *string    `json:"status"`
	Timestamp  *time.Time `json:"timestamp"`
	MethodType *string    `json:"method_type,omitempty"`
}

type event struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"`
	Timestamp time.Time       `json:"timestamp"`
	Payload   json.RawMessage `json:"payload"`
}

type eventResponse struct {
	Status    string `json:"status"`
	Partition int32  `json:"partition"`
	Offset    int64  `json:"offset"`
	Event     event  `json:"event"`
}

type healthResponse struct {
	Status bool `json:"status"`
}

type errorResponse struct {
	Error string `json:"error"`
}
