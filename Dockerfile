FROM alpine:3.22.0

WORKDIR /app
COPY ./tmp/app app
RUN chmod +x app

CMD ["./app"]
