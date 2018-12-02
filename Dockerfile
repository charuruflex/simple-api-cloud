FROM alpine:latest
EXPOSE 8000
ADD bin/api .
CMD ["./api"]