FROM golang:1.11-alpine AS builder
RUN apk add make git libc-dev gcc

WORKDIR /app
COPY . .

RUN go get -d -v ./...
RUN go build -a -ldflags '-w -extldflags "-static"' -o developer-gateway github.com/oasislabs/developer-gateway/cmd/gateway

FROM alpine as developer-gateway
ARG COMMIT_SHA
ARG BUILD_IMAGE_TAG
LABEL com.oasislabs.developer-gateway-commit-sha="${COMMIT_SHA}"
LABEL com.oasislabs.developer-gateway-build-image-tag="${BUILD_IMAGE_TAG}"
WORKDIR /
COPY --from=builder /app/cmd/gateway/config /config
COPY --from=builder /app/developer-gateway /developer-gateway
COPY --from=builder /app/mqueue/redis/redis.lua /redis.lua
CMD ["/developer-gateway"]
