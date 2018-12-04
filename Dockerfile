FROM alpine:latest
EXPOSE 8000
ADD bin/api .
ADD config.yml /etc/api/config.yml
CMD ["./api", "-config", "/etc/api/config.yml"]