FROM golang:1.13.6-alpine3.11 AS builder

RUN apk update && \
    apk add --no-cache git=2.24.1-r0 ca-certificates && \
    update-ca-certificates

# Create new user with limited privileges.
RUN adduser -D -g '' app

WORKDIR $HOME/src
COPY . .
RUN CGO_ENABLED=0 go build -o /go/bin/crawler-api github.com/emilyzhang/crawlr/cmd/api

# Create new lightweight container for binary file.
FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /go/bin/crawler-api /go/bin/crawler-api
COPY --from=builder /etc/passwd /etc/passwd
USER app

ENTRYPOINT ["/go/bin/crawler-api"]
