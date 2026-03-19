FROM golang:1.25-alpine AS builder
RUN apk add --no-cache git ca-certificates
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath -ldflags="-w -s -X main.version=${VERSION}" \
    -o /erp-p2p-cloud ./cmd/server

FROM alpine:3.19
RUN apk --no-cache add ca-certificates \
    && addgroup -g 1000 -S app && adduser -u 1000 -S app -G app
COPY --from=builder /erp-p2p-cloud /usr/local/bin/erp-p2p-cloud
USER app
EXPOSE 4001/tcp 4001/udp 8080 9090
ENTRYPOINT ["erp-p2p-cloud"]
