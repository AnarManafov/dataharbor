# Certificate Init Container - Generates self-signed certs for development

FROM alpine:3.22

RUN apk add --no-cache openssl

COPY docker/cert-init/generate-dev-certs.sh /usr/local/bin/generate-dev-certs.sh
RUN chmod +x /usr/local/bin/generate-dev-certs.sh

WORKDIR /certs

ENTRYPOINT ["/usr/local/bin/generate-dev-certs.sh"]
