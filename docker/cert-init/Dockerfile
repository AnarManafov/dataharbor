# Certificate Init Container - Generates self-signed certs for development

FROM alpine:3

RUN apk add --no-cache openssl dos2unix

COPY docker/cert-init/generate-dev-certs.sh /usr/local/bin/generate-dev-certs.sh
RUN dos2unix /usr/local/bin/generate-dev-certs.sh && \
    chmod +x /usr/local/bin/generate-dev-certs.sh

WORKDIR /certs

ENTRYPOINT ["/usr/local/bin/generate-dev-certs.sh"]
