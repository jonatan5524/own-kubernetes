FROM gcc:latest AS builder

WORKDIR /build

COPY . .

RUN gcc -o pause pause.c

ENTRYPOINT "./pause"