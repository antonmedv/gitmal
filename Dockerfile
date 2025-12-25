FROM golang:latest as builder

WORKDIR /go

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o gitmal .


FROM alpine

RUN apk add --no-cache git

COPY --from=builder /go/gitmal /bin/gitmal

WORKDIR /data

ENV COLORTERM=truecolor

ENTRYPOINT ["/bin/gitmal"]
