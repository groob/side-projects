FROM alpine:3.4

RUN apk --update add \
    ca-certificates 

COPY ./build/mac-loginwindow-png-linux-amd64 /mac-loginwindow-png
COPY css /css
COPY templates /templates

CMD ["/mac-loginwindow-png"]



