FROM golang:1.5

ENV GOPATH /go:/go/src/github.com/ahmetalpbalkan/wagl/Godeps/_workspace
ADD . /go/src/github.com/ahmetalpbalkan/wagl

WORKDIR /go/src/github.com/ahmetalpbalkan/wagl

RUN GOOS=linux GOARCH=amd64 \
	go install

RUN ["wagl", "--help"]

# 53: DNS
EXPOSE 53/udp

CMD ["wagl"]
