FROM golang:1.22
WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY *.go ./
RUN go mod download
RUN CGO_ENABLED=0 go build -o /aws-checker

EXPOSE 8080

CMD ["/aws-checker"]
