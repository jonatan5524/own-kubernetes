FROM golang:1.22-alpine AS builder

ARG target
ARG dir

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o $target ./$dir

FROM alpine:3.20.2

ARG target

COPY --from=builder "/build/$target" "./"

ENV TARGET_SH=$target
ENTRYPOINT "./$TARGET_SH"