FROM nginx:1.27-alpine

COPY deployments/docker/web.default.conf /etc/nginx/conf.d/default.conf
COPY web /usr/share/nginx/html
EXPOSE 80
