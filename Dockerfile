FROM debian:bookworm-slim

RUN apt update && apt install ca-certificates -y

COPY build/appetizer .
COPY *.html .

ENTRYPOINT [ "./appetizer" ]