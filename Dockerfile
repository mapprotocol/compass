FROM golang:1.20.4-alpine as builder

RUN mkdir /compass

COPY . /compass

RUN apk add --no-cache musl-dev gcc

RUN cd /compass && cd cmd/compass && GOOS=linux go build -o ../../build/compass && cp ../../eth2/eth2-proof ../../build/

FROM alpine as prod

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR  /root

COPY --from=builder /compass/build/compass /root/compass
COPY --from=builder /compass/build/eth2-proof /root/eth2-proof

RUN chmod +x /root/eth2-proof && chmod +x /root/compass

ENTRYPOINT ["/root/compass"]