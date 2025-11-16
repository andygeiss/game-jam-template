FROM scratch
ADD bin/server-linux-amd64 /server
ENV PORT=8080
ENTRYPOINT [ "/server" ]
