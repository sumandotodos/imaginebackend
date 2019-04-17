FROM golang:latest AS builder
WORKDIR /app
COPY src/server.go .
RUN go get github.com/gorilla/mux
RUN go get go.mongodb.org/mongo-driver/bson
RUN go get go.mongodb.org/mongo-driver/mongo
RUN go get go.mongodb.org/mongo-driver/mongo/options
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server .
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/server .
CMD ["./server"]
EXPOSE 9911
