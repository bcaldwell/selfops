FROM golang:1.13 AS builder

# Copy the code from the host and compile it
# ENV GO111MODULE=on
WORKDIR $GOPATH/src/github.com/bcaldwell/selfops
# COPY go.* ./
# RUN go mod download

COPY . ./
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix nocgo -o /selfops .

FROM alpine
RUN apk update && apk add ca-certificates && rm -rf /var/cache/apk/*
WORKDIR /selfops
COPY --from=builder /selfops ./
ENTRYPOINT ["./selfops"]
