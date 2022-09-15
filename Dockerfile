FROM golang:1.17-buster as builder

ENV GO111MODULE "on"

ARG BUILD_VER

WORKDIR /usr/local/go/src/app
COPY . .
RUN go mod download
ENV GOOS=linux
ENV GOARCH=amd64
ENV CGO_ENABLED=0

RUN go build \
  -v \
  -ldflags "-w -s -X 'main.BuildDatetime=$(date --iso-8601=seconds)' -X 'main.BuildVer=${BUILD_VER}'" \
  -o server \
  ./main.go

FROM alpine:3.13
WORKDIR /app
COPY --from=builder /usr/local/go/src/app/server /app/
RUN apk add curl jq iproute2 bind-tools --no-cache
ENTRYPOINT ["/app/server"]
LABEL maintainer="flsixtyfour@gmail.com"
LABEL org.label-schema.vcs-url="https://github.com/fl64/http-keepalive-test"
LABEL org.label-schema.docker.cmd="docker run --rm fl64/http-keepalive-test:latest"
