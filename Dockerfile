FROM golang:1.26-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o /csl-bench ./cmd/csl-bench

FROM alpine:3.21

COPY --from=builder /csl-bench /usr/local/bin/csl-bench
ENTRYPOINT ["csl-bench"]
