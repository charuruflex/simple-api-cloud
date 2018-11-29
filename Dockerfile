# # build
FROM golang:alpine as builder
RUN apk -U add git
# RUN mkdir -p /go/src/app/build /go/src/app
WORKDIR /go/src/api
# COPY . .
ADD . /go/src/api
RUN go get
RUN go install api

# # deployment
FROM alpine:latest
# ARG PORT
# ARG SIZE
# ENV PORT ${PORT}
# ENV SIZE ${SIZE}
# RUN mkdir /app
EXPOSE 8000
COPY --from=builder /go/bin/api .
CMD ["./api"]

# FROM alpine:latest

# # RUN apk -U add ca-certificates

# EXPOSE 8080

# ADD api /bin/api
# # ADD config.yml /etc/news/config.yml

# CMD ["api"]