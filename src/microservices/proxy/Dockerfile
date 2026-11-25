FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod *.go ./
RUN CGO_ENABLED=0 go build -o /proxy-service .

FROM alpine:latest
COPY --from=builder /proxy-service /usr/local/bin/proxy-service
USER 65532:65532
EXPOSE 8000
ENTRYPOINT ["/usr/local/bin/proxy-service"]
