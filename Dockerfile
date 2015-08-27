FROM golang:1.5

ENV GOPATH /go
RUN mkdir -p /go/src/wagl
ADD . /go/src/wagl
WORKDIR /go/src/wagl

RUN go get -d -v
RUN GOOS=linux GOARCH=amd64 \
	go build -o /wagl
ENTRYPOINT ["/wagl"]

EXPOSE 53/udp
