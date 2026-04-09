FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o surf-recommender ./cmd/surf-recommender

FROM alpine:3.21
# tzdata — для корректной работы Europe/Lisbon в планировщике
# ca-certificates — для HTTPS-запросов к Stormglass и Anthropic
RUN apk add --no-cache tzdata ca-certificates
WORKDIR /app
COPY --from=builder /app/surf-recommender .
COPY config/ config/
ENTRYPOINT ["./surf-recommender"]
CMD ["--daemon"]
