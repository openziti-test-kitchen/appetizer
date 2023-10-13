FROM debian:bookworm-slim

RUN apt update && apt install ca-certificates -y

COPY build/appetizer /
COPY http_content /http_content

WORKDIR /
ENTRYPOINT [ "/appetizer" ]