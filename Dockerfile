FROM golang:1.24-alpine AS goexec-builder
LABEL builder="true"

WORKDIR /go/src/github.com/FalconOpsLLC/goexec

COPY . .

ARG CGO_ENABLED=0

RUN go mod download
RUN go build -ldflags="-s -w" -trimpath -o /go/bin/goexec

FROM scratch
COPY --from="goexec-builder" /go/bin/goexec /goexec

WORKDIR /io
VOLUME ["/io"]
ENTRYPOINT ["/goexec"]
