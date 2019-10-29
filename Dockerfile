FROM golang:1.13 AS builder

WORKDIR $GOPATH/src/github.com/bcaldwell/selfops

COPY . ./
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix nocgo -o /selfops .

FROM alpine
RUN apk update && apk add ca-certificates && rm -rf /var/cache/apk/*
WORKDIR /selfops
COPY --from=builder /selfops ./
ENTRYPOINT ["./selfops"]
