FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
RUN CGO_ENABLED=0 go build -o /events-service .

FROM alpine:latest
COPY --from=builder /events-service /usr/local/bin/events-service
USER 65532:65532
EXPOSE 8082
ENTRYPOINT ["/usr/local/bin/events-service"]
