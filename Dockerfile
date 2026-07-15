FROM golang:1.26.3-alpine AS builder

ARG VERSION
ARG COMMIT_ID
ARG BUILD_TIME

RUN mkdir /compass

COPY . /compass

RUN apk add --no-cache musl-dev gcc linux-headers git

RUN cd /compass && \
    VERSION="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo dev)}" && \
    COMMIT_ID="${COMMIT_ID:-$(git rev-parse --short HEAD 2>/dev/null || echo unknown)}" && \
    BUILD_TIME="${BUILD_TIME:-$(date -u +"%Y-%m-%dT%H:%M:%SZ")}" && \
    cd cmd/compass && \
    GOOS=linux go build -ldflags "-X main.Version=${VERSION} -X main.CommitID=${COMMIT_ID} -X main.BuildTime=${BUILD_TIME}" -o ../../build/compass

FROM alpine AS prod

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ=Asia/Shanghai

WORKDIR  /root

COPY --from=builder /compass/build/compass /root/compass

RUN chmod +x /root/compass

ENTRYPOINT ["/root/compass"]
