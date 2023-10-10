FROM scratch

COPY build/appetizer .
COPY *.html .

ENTRYPOINT [ "./appetizer" ]