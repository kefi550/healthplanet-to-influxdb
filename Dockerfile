FROM golang:1.22-bookworm AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
COPY cmd ./cmd

WORKDIR /app/cmd
RUN CGO_ENABLED=0 GOOS=linux go build -o /healthplanet-to-influxdb

CMD ["/healthplanet-to-influxdb"]
