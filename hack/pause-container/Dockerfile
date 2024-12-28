FROM gcc:latest AS builder

WORKDIR /build

COPY . .

RUN gcc -o pause pause.c

FROM scratch

COPY --from=builder /build/pause /pause

USER 65535:65535

ENTRYPOINT ["/pause"]