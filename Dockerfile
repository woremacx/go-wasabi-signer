FROM woremacx/golang:1.20-lunar AS builder

COPY ./ /build
WORKDIR /build

RUN go mod tidy -v

RUN go build -v


FROM ubuntu:23.04

# ca-certificates: to prevent "tls: failed to verify certificate: x509: certificate signed by unknown authority"
RUN set -eux; \
	apt-get update; \
	apt-get install -y --no-install-recommends \
		ca-certificates \
	; \
	rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /build/go-wasabi-signer /app/go-wasabi-signer

EXPOSE 8080

USER nobody:nogroup

CMD ["/app/go-wasabi-signer"]

LABEL org.opencontainers.image.source https://github.com/woremacx/go-wasabi-signer
