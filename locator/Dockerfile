# Start with a full-fledged golang image, but strip it from the final image.
FROM golang:1.11.5-alpine as builder


COPY . /go/src/github.com/gdotgordon/locator-demo/locator

WORKDIR /go/src/github.com/gdotgordon/locator-demo/locator

RUN go build -v

FROM alpine:latest

WORKDIR /root/

# Make a significantly slimmed-down final result.
COPY --from=builder /go/src/github.com/gdotgordon/locator-demo/locator .

# That's me!
LABEL maintainer="Gary Gordon <gagordon12@gmail.com>"

CMD ["./locator"]
