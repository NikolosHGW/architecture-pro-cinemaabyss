package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"strconv"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
)

const (
	movieTopic     = "movie-events"
	userTopic      = "user-events"
	paymentTopic   = "payment-events"
	maxEventBytes  = 256 << 10
	publishTimeout = 4 * time.Second
)

type eventProducer interface {
	ProduceSync(context.Context, ...*kgo.Record) kgo.ProduceResults
}

type eventHandlers struct {
	producer eventProducer
}

func newHandler(producer eventProducer) http.Handler {
	handlers := eventHandlers{producer: producer}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/events/health", healthHandler)
	mux.HandleFunc("POST /api/events/movie", handlers.createMovieEventHandler)
	mux.HandleFunc("POST /api/events/payment", handlers.createPaymentEventHandler)
	mux.HandleFunc("POST /api/events/user", handlers.createUserEventHandler)
	return mux
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, healthResponse{Status: true})
}

func (h eventHandlers) createMovieEventHandler(w http.ResponseWriter, r *http.Request) {
	var input movieEvent
	payload, err := readEventPayload(w, r, &input,
		[]string{"movie_id", "title", "action"},
		[]string{"user_id", "rating", "genres", "description"},
	)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	h.publishEvent(w, r, movieTopic, "movie", *input.MovieID, time.Now().UTC(), payload)
}

func (h eventHandlers) createPaymentEventHandler(w http.ResponseWriter, r *http.Request) {
	var input paymentEvent
	payload, err := readEventPayload(w, r, &input,
		[]string{"payment_id", "user_id", "amount", "status", "timestamp"},
		[]string{"method_type"},
	)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	h.publishEvent(w, r, paymentTopic, "payment", *input.PaymentID, *input.Timestamp, payload)
}

func (h eventHandlers) createUserEventHandler(w http.ResponseWriter, r *http.Request) {
	var input userEvent
	payload, err := readEventPayload(w, r, &input,
		[]string{"user_id", "action", "timestamp"},
		[]string{"username", "email"},
	)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	h.publishEvent(w, r, userTopic, "user", *input.UserID, *input.Timestamp, payload)
}

func readEventPayload(w http.ResponseWriter, r *http.Request, input any, required, optional []string) (json.RawMessage, error) {
	if header := r.Header.Get("Content-Type"); header != "" {
		mediaType, _, err := mime.ParseMediaType(header)
		if err != nil || mediaType != "application/json" {
			return nil, errors.New("Content-Type must be application/json")
		}
	}

	payload, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxEventBytes))
	if err != nil {
		return nil, errors.New("invalid request body or body exceeds 256 KiB")
	}
	if err := decodePayload(payload, input, required, optional); err != nil {
		return nil, err
	}
	return payload, nil
}

func (h eventHandlers) publishEvent(w http.ResponseWriter, r *http.Request, topic, eventType string, key int64, timestamp time.Time, payload json.RawMessage) {
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		slog.Error("generate event ID", "type", eventType, "topic", topic, "error", err)
		writeError(w, http.StatusInternalServerError, "could not create event")
		return
	}
	message := event{ID: hex.EncodeToString(id[:]), Type: eventType, Timestamp: timestamp, Payload: payload}
	value, err := json.Marshal(message)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not encode event")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), publishTimeout)
	defer cancel()
	record, err := h.producer.ProduceSync(ctx, &kgo.Record{
		Topic: topic,
		Key:   strconv.AppendInt(nil, key, 10),
		Value: value,
	}).First()
	if err != nil {
		slog.Error("publish event", "event_id", message.ID, "type", eventType, "topic", topic, "error", err)
		writeError(w, http.StatusInternalServerError, "could not publish event")
		return
	}
	writeJSON(w, http.StatusCreated, eventResponse{
		Status: "success", Partition: record.Partition, Offset: record.Offset, Event: message,
	})
}

func decodePayload(payload []byte, input any, required, optional []string) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(payload, &fields); err != nil || fields == nil {
		return errors.New("request body must be a JSON object")
	}

	known := make(map[string]json.RawMessage, len(required)+len(optional))
	for _, name := range required {
		value := fields[name]
		if value == nil || bytes.Equal(value, []byte("null")) {
			return fmt.Errorf("field %s is required", name)
		}
		known[name] = value
	}
	for _, name := range optional {
		if value, exists := fields[name]; exists {
			if bytes.Equal(value, []byte("null")) {
				return fmt.Errorf("field %s must not be null", name)
			}
			known[name] = value
		}
	}
	validated, err := json.Marshal(known)
	if err != nil {
		return errors.New("invalid JSON fields")
	}
	if err := json.Unmarshal(validated, input); err != nil {
		return errors.New("invalid field type or timestamp")
	}
	return nil
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Error: message})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Error("write HTTP response", "status", status, "error", err)
	}
}
