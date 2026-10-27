# nginx menerima HTTP lalu meneruskan ke PHP-FPM lewat protokol FastCGI (unix socket).
# PHP tidak punya HTTP server produksi bawaan; Go/Rust/Node/Java melayani HTTP sendiri.
worker_processes __NGINX_WORKERS__;
pid /tmp/nginx.pid;
error_log /dev/stderr warn;

events {
    worker_connections 4096;
}

http {
    access_log off;
    server_tokens off;
    client_max_body_size 1m;
    keepalive_timeout 65s;
    keepalive_requests 100000;

    client_body_temp_path /tmp/nginx-body;
    fastcgi_temp_path /tmp/nginx-fastcgi;
    proxy_temp_path /tmp/nginx-proxy;
    uwsgi_temp_path /tmp/nginx-uwsgi;
    scgi_temp_path /tmp/nginx-scgi;

    server {
        listen 8080;
        root /app/public;

        location = /metrics {
            include /etc/nginx/fastcgi_params;
            fastcgi_param SCRIPT_FILENAME /app/public/index.php;
            fastcgi_pass unix:/tmp/php-fpm-ops.sock;
        }

        location / {
            include /etc/nginx/fastcgi_params;
            fastcgi_param SCRIPT_FILENAME /app/public/index.php;
            fastcgi_pass unix:/tmp/php-fpm.sock;
        }
    }
}
