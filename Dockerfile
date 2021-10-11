FROM golang:1.17 AS builder

WORKDIR $GOPATH/src/github.com/bcaldwell/selfops

COPY . ./
RUN go build -o /selfops .

FROM alpine
RUN apk update && apk add ca-certificates tzdata && rm -rf /var/cache/apk/*
WORKDIR /selfops
COPY --from=builder /selfops ./
ENTRYPOINT ["./selfops"]
