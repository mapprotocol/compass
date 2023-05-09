FROM golang:1.20.4 as builder

ENV TZ Asia/Shanghai

RUN mkdir /compass

COPY . /compass

RUN cd /compass && make build

WORKDIR /compass/build

ENTRYPOINT ["/compass/build/compass"]

#FROM golang:alpine as prod
#
#RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
#ENV TZ Asia/Shanghai
#
#WORKDIR  /home
#
#COPY --from=builder /compass/build/compass /home/compass
#COPY --from=builder /compass/build/eth2-proof /home/eth2-proof
#
#RUN chmod +x /home/eth2-proof

CMD ["ls", "-alh", "/home/"]
