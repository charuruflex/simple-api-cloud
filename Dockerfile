FROM alpine:latest
EXPOSE 8000
ADD bin/api .
ADD config.yml /etc/news/config.yml
CMD ["./api", "-config", "/etc/news/config.yml"]