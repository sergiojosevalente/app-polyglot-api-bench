#!/bin/sh
set -e

# nginx tidak bisa membaca env, jadi jumlah worker disisipkan ke config saat start.
sed "s/__NGINX_WORKERS__/${NGINX_WORKERS:-1}/" /etc/nginx/nginx.conf.tpl > /tmp/nginx.conf
nginx -c /tmp/nginx.conf

# php-fpm menjadi PID 1: kalau mati, container ikut berhenti dan di-restart Docker.
exec php-fpm -F
