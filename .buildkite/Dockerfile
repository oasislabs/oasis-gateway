FROM golang:1.14-alpine AS builder

# Set GOPROXY to accept go proxy as a build arg and set as an environment
# variable. By default this value is empty and not use a go proxy.
ARG GOPROXY=

RUN apk add make git libc-dev gcc

WORKDIR /app
COPY . .

RUN go get -d -v ./...
RUN go build -a -ldflags '-w -extldflags "-static"' -o oasis-gateway github.com/oasislabs/oasis-gateway/cmd/gateway

FROM alpine as oasis-gateway
ARG COMMIT_SHA
ARG BUILD_IMAGE_TAG
LABEL com.oasislabs.oasis-gateway-commit-sha="${COMMIT_SHA}"
LABEL com.oasislabs.oasis-gateway-build-image-tag="${BUILD_IMAGE_TAG}"
WORKDIR /
COPY --from=builder /app/cmd/gateway/config /config
COPY --from=builder /app/oasis-gateway /oasis-gateway
COPY --from=builder /app/mqueue/redis/redis.lua /redis.lua
CMD ["/oasis-gateway"]
