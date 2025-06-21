FROM debian
VOLUME /data
RUN mkdir -p /etc/goa4web-bookmarks
COPY example-config.json /etc/goa4web-bookmarks/config.json
COPY example.env /etc/goa4web-bookmarks/goa4web-bookmarks.env
ENV DB_CONNECTION_PROVIDER=sqlite3
ENV DB_CONNECTION_STRING="file:/data/a4webbookmarks.db?_loc=auto"
# ENV DB_CONNECTION_PROVIDER=mysql
# ENV DB_CONNECTION_STRING="a4webmb:......@tcp(.....:3306)/a4webbm?parseTime=true"
EXPOSE 8080
EXPOSE 8443
COPY a4webbmws /bin/a4webbmws
RUN apt-get update && apt-get install -y \
  libsqlite3-0 \
  ca-certificates \
  && rm -rf /var/lib/apt/lists/* && update-ca-certificates
ENV PATH=/bin
ENTRYPOINT ["a4webbmws"]
