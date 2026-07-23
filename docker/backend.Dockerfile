FROM golang:1.23-alpine AS builder
WORKDIR /src
COPY backend/go.mod backend/go.sum* ./
RUN go mod download
COPY backend/ .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

FROM alpine:3.21
RUN apk add --no-cache ca-certificates wget && addgroup -S app && adduser -S app -G app
WORKDIR /app
COPY --from=builder /out/api /usr/local/bin/api
RUN mkdir -p /app/storage && chown -R app:app /app
USER app
EXPOSE 8080
CMD ["api"]
