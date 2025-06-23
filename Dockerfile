FROM alpine
VOLUME /data
VOLUME /etc/goa4web-bookmarks
ENV EXTERNAL_URL=http://localhost:8080
ENV OAUTH2_CLIENT_ID=""
ENV OAUTH2_SECRET=""
ENV OAUTH2_AUTH_URL=""
ENV OAUTH2_TOKEN_URL=""
ENV GBM_CSS_COLUMNS=""
ENV GBM_NAMESPACE=""
ENV GBM_TITLE=""
ENV FAVICON_CACHE_DIR=/data/favicons
ENV FAVICON_CACHE_SIZE=20971520
ENV GBM_NO_FOOTER=""
ENV DB_CONNECTION_PROVIDER=sqlite3
ENV DB_CONNECTION_STRING="file:/data/a4webbookmarks.db?_loc=auto"
ENV GOBM_ENV_FILE=/etc/goa4web-bookmarks/goa4web-bookmarks.env
ENV GOBM_CONFIG_FILE=/etc/goa4web-bookmarks/config.json
EXPOSE 8080
EXPOSE 8443
COPY a4webbmws /bin/a4webbmws
RUN apk add --no-cache ca-certificates libsqlite3 && update-ca-certificates
ENV PATH=/bin
ENTRYPOINT ["a4webbmws"]
