FROM woremacx/golang:1.20-lunar AS builder

COPY ./ /build
WORKDIR /build

RUN go mod tidy -v

RUN go build -v


FROM ubuntu:23.04


WORKDIR /app

COPY --from=builder /build/go-wasabi-signer /app/go-wasabi-signer

EXPOSE 8080

USER nobody:nogroup

CMD ["/app/go-wasabi-signer"]

LABEL org.opencontainers.image.source https://github.com/woremacx/go-wasabi-signer
