FROM golang:1.14 AS builder

RUN apt-get -qq update && apt-get -yqq install upx

ENV GO111MODULE=on \
  CGO_ENABLED=0 \
  GOOS=linux \
  GOARCH=amd64

WORKDIR /src

COPY . .

RUN go build \
  -trimpath \
  -ldflags "-s -w -extldflags '-static'" \
  -installsuffix cgo \
  -tags netgo \
  -o /app/bot \
  ./cmd/bloopsbot-srv

RUN strip /app/bot
RUN upx -q -9 /app/bot

RUN mkdir /data

FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/bot /app/bot
COPY --from=builder /data /data

VOLUME /data

ENTRYPOINT ["/app/bot"]
