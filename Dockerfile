FROM golang:1.20.4 as builder

ENV TZ Asia/Shanghai

RUN mkdir /compass

COPY . /compass

RUN cd /compass && make build

WORKDIR /compass/build

ENTRYPOINT ["/compass/build/compass"]

CMD ["ls", "-alh", "/home/"]
