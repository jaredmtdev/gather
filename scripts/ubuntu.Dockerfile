FROM ubuntu:25.04@sha256:728785b59223d755e3e5c5af178fab1be7031f3522c5ccd7a0b32b80d8248123

ARG GO_VERSION=1.25.1
ARG TARGETARCH=arm64
ARG GO_SHA=65a3e34fb2126f55b34e1edfc709121660e1be2dee6bdf405fc399a63a95a87d

RUN apt-get update && apt-get install -y --no-install-recommends \
      ca-certificates curl git build-essential \
    && rm -rf /var/lib/apt/lists/* \
 && curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-${TARGETARCH}.tar.gz" -o /tmp/go.tgz \
 && echo "${GO_SHA} /tmp/go.tgz" | sha256sum --check \
 && tar -C /usr/local -xzf /tmp/go.tgz \
 && rm /tmp/go.tgz

ENV GOROOT=/usr/local/go
ENV GOPATH=/go
ENV PATH="${GOROOT}/bin:${GOPATH}/bin:${PATH}"
ENV CGO_ENABLED=1

#RUN go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@43d03392d7dc3746fa776dbddd66dfcccff70651 # v2.4.0
#RUN go install golang.org/x/vuln/cmd/govulncheck@d1f380186385b4f64e00313f31743df8e4b89a77

RUN go version
