FROM golang:1.23.0-alpine3.20 AS builder

WORKDIR $GOPATH/src/github.com/bcaldwell/selfops

COPY . ./
RUN go build -o /selfops .

FROM alpine:3.20.2
RUN apk update && apk add ca-certificates tzdata && rm -rf /var/cache/apk/*
WORKDIR /selfops
COPY --from=builder /selfops ./
ENTRYPOINT ["./selfops"]
