FROM golang:1.11.5-alpine as builder

LABEL maintainer="Gary Gordon <gagordon12@gmail.com>"

RUN apk add git

COPY . /go/src/github.com/gdotgordon/locator-demo

RUN go get -v -d ./...

WORKDIR /go/src/github.com/gdotgordon/locator-demo

RUN go build -v

FROM alpine:latest

WORKDIR /root/

COPY --from=builder /go/src/github.com/gdotgordon/locator-demo .

CMD ["./locator-demo"]
