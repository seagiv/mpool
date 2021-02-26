FROM golang as builder

WORKDIR /go/src/dbgw/

COPY /github.com/seagiv/sfw /go/src/github.com/seagiv/sfw
COPY /github.com/seagiv/dbgw /go/src/dbgw

RUN go get -d
RUN go build -o /bin/dbgw

FROM debian:stable-slim
COPY --from=builder /bin/dbgw /bin/dbgw

COPY --from=builder /go/src/dbgw/dbgw.conf /etc/
COPY --from=builder /go/src/dbgw/certs /usr/share/dbgw/certs

EXPOSE 8223
ENTRYPOINT ["/bin/dbgw"]