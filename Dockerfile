# Start with a full-fledged golang image, but strip it from the final image.
FROM golang:1.11.5-alpine as builder

# That's me!
LABEL maintainer="Gary Gordon <gagordon12@gmail.com>"

# Yep, we're even going to do the 'go get' for you,so we need git.
RUN apk add git

COPY . /go/src/github.com/gdotgordon/locator-demo

RUN go get -v -d ./...

WORKDIR /go/src/github.com/gdotgordon/locator-demo

RUN go build -v

FROM alpine:latest

WORKDIR /root/

# Make a significantly slimmed-down final result.
COPY --from=builder /go/src/github.com/gdotgordon/locator-demo .

CMD ["./locator-demo"]
