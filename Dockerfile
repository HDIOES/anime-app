FROM debian:stretch
COPY anime-app ./
COPY settings.json ./
COPY migrations/* ./migrations/ 
ENTRYPOINT ["./anime-app"]
