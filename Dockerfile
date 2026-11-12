ARG GO_IMAGE=golang:1.24

FROM --platform=linux/amd64 ${GO_IMAGE} AS build

WORKDIR /app

COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal

ARG VERSION=dev 

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w -X main.version=${VERSION}" -o /sealix ./cmd/sealix

FROM scratch

ARG SOURCE_REPO=https://github.com/FityatulhaqRosyidi/sealix

LABEL org.opencontainers.image.source=${SOURCE_REPO}

COPY --from=build /sealix /sealix 

ENTRYPOINT [ "/sealix" ]
