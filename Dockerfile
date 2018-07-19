FROM golang:1.10 AS builder

# Download and install the latest release of dep
# ADD https://github.com/golang/dep/releases/download/v0.4.1/dep-linux-amd64 /usr/bin/dep
# RUN chmod +x /usr/bin/dep

# Copy the code from the host and compile it
WORKDIR $GOPATH/src/github.com/bcaldwell/ynab-influx-importer
# COPY Gopkg.toml Gopkg.lock ./
# RUN dep ensure --vendor-only
COPY . ./
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix nocgo -o /ynab-influx-importer .

FROM alpine
RUN apk update && apk add ca-certificates && rm -rf /var/cache/apk/*
WORKDIR /ynab-influx-importer
COPY --from=builder /ynab-influx-importer ./
ENTRYPOINT ["./ynab-influx-importer"]