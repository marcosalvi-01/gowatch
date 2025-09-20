FROM alpine:3.22.1

WORKDIR /gowatch

COPY ./tmp/gowatch gowatch

RUN adduser -D -u 1000 appuser \
    && mkdir -p /var/lib/gowatch \
    && chown -R appuser:appuser /gowatch /var/lib/gowatch

HEALTHCHECK --interval=30s --timeout=5s \
  CMD wget -qO- http://localhost:8080/api/v1/health || exit 1

USER appuser

EXPOSE 8080

CMD ["./gowatch"]
