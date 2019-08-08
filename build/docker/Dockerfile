FROM golang:alpine

RUN apk add --no-cache git tar gzip curl

RUN curl -L https://github.com/concourse/concourse/releases/download/v5.4.0/fly-5.4.0-linux-amd64.tgz | tar zx

RUN chmod +x fly

RUN mv ./fly /usr/bin

WORKDIR /concourse-job-log-resource

COPY . .

RUN mkdir -p /opt/resource

RUN go build -o /opt/resource/check ./cmd/check/main.go
RUN go build -o /opt/resource/in ./cmd/in/main.go
RUN go build -o /opt/resource/out ./cmd/out/main.go

