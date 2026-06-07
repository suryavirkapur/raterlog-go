FROM golang:1.26-alpine AS build

WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/raterlog ./cmd/raterlog

FROM alpine:3.22
RUN adduser -D -H raterlog
USER raterlog
COPY --from=build /out/raterlog /usr/local/bin/raterlog
EXPOSE 8080
ENTRYPOINT ["raterlog"]
