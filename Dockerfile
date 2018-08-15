FROM alpine:3.8
LABEL maintainer="Matthias Luedtke (matthiasluedtke)"

RUN apk add --no-cache ca-certificates
# Fixes 'Get https://github.com/: x509: failed to
# load system roots and no roots provided'

# https://docs.docker.com/develop/develop-images/dockerfile_best-practices/#add-or-copy
COPY bin/linux_amd64/faqaas /var/www/faqaas
COPY admin/templates/ /var/www/admin/templates
COPY public/ /var/www/public

EXPOSE 8080
ENV PORT 8080

ENV DATABASE_URL postgres://postgres:@localhost/faqaas?sslmode=disable
ENV SUPPORTED_LOCALES en,de,fr,es,it,nl,pt,pt-BR,da,sv,no,ru,ar,zh
ENV JWT_KEY=secret
ENV ADMIN_PASSWORD '$2a$12$AfzzMbT65vzPrF0DegdrZO39rHe.aABxMM6GQfKihkv4xh/YW.RKm'
ENV HTTP_ALLOWED true
ENV API_KEY deadbeef

RUN find /var/www
WORKDIR /var/www
CMD ["./faqaas"]
