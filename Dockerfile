FROM debian:bookworm-slim

RUN apt update && apt install ca-certificates -y

COPY build/appetizer .
COPY *.html .
COPY http_content http_content

ENTRYPOINT [ "./appetizer" ]