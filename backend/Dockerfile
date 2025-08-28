FROM alpine:latest
WORKDIR /app
COPY app/bin/app /app/
COPY app/web/static/ /app/web/static/
EXPOSE 8080
ENTRYPOINT ["/app/app"]
