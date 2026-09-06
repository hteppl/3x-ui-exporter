FROM golang:1.27 AS builder

ARG TARGETPLATFORM
ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH
ARG GIT_TAG
ARG GIT_COMMIT

ENV CGO_ENABLED=0
ENV GO111MODULE=on
ENV GOOS=${TARGETOS}
ENV GOARCH=${TARGETARCH}

WORKDIR /build

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

RUN --mount=type=cache,target=/root/.cache/go-build \
    go build -ldflags="-w -s -X main.version=${GIT_TAG} -X main.commit=${GIT_COMMIT}" \
    -trimpath \
    -o /usr/bin/x-ui-exporter . && \
    go clean -modcache

FROM gcr.io/distroless/static-debian12:nonroot

LABEL org.opencontainers.image.source=https://github.com/hteppl/x-ui-exporter

USER nonroot:nonroot

WORKDIR /
COPY --from=builder --chown=nonroot:nonroot /usr/bin/x-ui-exporter /x-ui-exporter

ENV PATH="/:${PATH}"

ENTRYPOINT ["/x-ui-exporter"]
