FROM debian:stretch
COPY anime-app settings.json migrations/ ./
ENTRYPOINT ["./anime-app"]
