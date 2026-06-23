# Stage 1: Build
FROM golang:1.24-alpine AS builder
RUN apk add --no-cache ca-certificates
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /headroom ./cmd/headroom/

# Stage 2: Runtime
FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /headroom /headroom
EXPOSE 18787
ENTRYPOINT ["/headroom"]
CMD ["proxy", "--port=18787"]
