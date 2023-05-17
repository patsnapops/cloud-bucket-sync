FROM golang:1.20 AS builder
LABEL maintainer="ops<ops@patsnap.com>"

ENV GOPATH /go
ENV GOROOT /usr/local/go
ENV GOPROXY https://goproxy.cn,direct
ENV BUILD_DIR ${GOPATH}/src/cbs
ENV CGO_ENABLED 0

# Print go version
RUN echo "GOROOT is ${GOROOT}"
RUN echo "GOPATH is ${GOPATH}"
RUN ${GOROOT}/bin/go version

# Build
WORKDIR ${BUILD_DIR}
COPY . ${BUILD_DIR}
RUN go mod tidy && go build && mv cbs /usr/bin/cbs

# Stage2
FROM centos7:latest

ENV TZ "Asia/Shanghai"

COPY --from=builder /usr/bin/cbs /bin/cbs
COPY --from=builder /go/src/cbs/entrypoint.sh /root/entrypoint.sh

WORKDIR /root

RUN chmod +x /root/entrypoint.sh 

USER root

ENTRYPOINT /root/entrypoint.sh