FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/control-plane ./cmd/control-plane

FROM alpine:3.21
RUN addgroup -S app && adduser -S -G app app
COPY --from=build /out/control-plane /usr/local/bin/control-plane
USER app
EXPOSE 45175
ENTRYPOINT ["/usr/local/bin/control-plane"]
