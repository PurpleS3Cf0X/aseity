FROM golang:1.23-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w -X github.com/jeanpaul/aseity/pkg/version.Version=docker" -o /aseity ./cmd/aseity

FROM alpine:3.20
RUN apk add --no-cache git bash curl
COPY --from=builder /aseity /usr/local/bin/aseity
WORKDIR /workspace
ENTRYPOINT ["aseity"]
