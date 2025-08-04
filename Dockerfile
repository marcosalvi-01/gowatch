FROM alpine:3.22.0

WORKDIR /gowatch
COPY ./tmp/gowatch gowatch
RUN chmod +x gowatch

CMD ["./gowatch"]
