ARG GO_BASE_IMAGE=docker.io/library/golang:1.25-alpine
ARG RUNTIME_BASE_IMAGE=docker.io/library/alpine:3.21
FROM ${GO_BASE_IMAGE} AS build
WORKDIR /src
ARG GOPROXY=https://goproxy.cn,direct
ENV GOPROXY=${GOPROXY}
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/control-plane ./cmd/control-plane

FROM ${RUNTIME_BASE_IMAGE}
RUN addgroup -S app && adduser -S -G app app
RUN mkdir -p /var/lib/agent-notify && chown -R app:app /var/lib/agent-notify
COPY --from=build /out/control-plane /usr/local/bin/control-plane
USER app
EXPOSE 45175
ENTRYPOINT ["/usr/local/bin/control-plane"]
