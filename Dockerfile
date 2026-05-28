FROM golang:1.26.3-alpine AS builder

RUN mkdir /compass

COPY . /compass

RUN apk add --no-cache musl-dev gcc linux-headers

RUN cd /compass && cd cmd/compass && GOOS=linux go build -o ../../build/compass

FROM alpine AS prod

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ=Asia/Shanghai

WORKDIR  /root

COPY --from=builder /compass/build/compass /root/compass

RUN chmod +x /root/compass

ENTRYPOINT ["/root/compass"]
