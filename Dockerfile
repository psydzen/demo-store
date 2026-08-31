FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/quiz ./cmd/quiz

FROM alpine:3.21
RUN adduser -D -u 10001 app
COPY --from=build /out/quiz /usr/local/bin/quiz
USER app
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/quiz"]
