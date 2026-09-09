FROM nginx:stable-alpine@sha256:dc5069ad14f19660b141b21236140b91656bf89bbc3e2417c70ae650cd66104c

COPY docker/default.conf /etc/nginx/conf.d/default.conf
COPY *.html showcase_data.json /usr/share/nginx/html/
COPY assets/ /usr/share/nginx/html/assets/

EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -q -O /dev/null http://127.0.0.1:8080/healthz || exit 1
