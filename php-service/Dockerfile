# PHP diinterpretasi, jadi tidak ada "build stage -> binary" seperti Go/Rust:
# image runtime harus membawa interpreter PHP + nginx.
FROM php:8.4-fpm-alpine

RUN apk add --no-cache nginx \
    && apk add --no-cache --virtual .build-deps $PHPIZE_DEPS \
    && pecl install apcu \
    && docker-php-ext-enable apcu \
    && docker-php-ext-install opcache \
    && apk del .build-deps \
    # access log FPM per request ke stderr = overhead; matikan
    && sed -i '/access.log/d' /usr/local/etc/php-fpm.d/docker.conf

# Composer: dependency manager + autoloader standar PHP (PSR-4).
COPY --from=composer:2 /usr/bin/composer /usr/bin/composer
WORKDIR /app
COPY composer.json ./
COPY src ./src
COPY public ./public
# classmap-authoritative: autoloader tidak pernah mencari file di disk (lebih cepat).
RUN composer dump-autoload --optimize --classmap-authoritative --no-dev \
    && rm /usr/bin/composer

COPY docker/php.ini /usr/local/etc/php/conf.d/zz-bench.ini
COPY docker/fpm-pools.conf /usr/local/etc/php-fpm.d/zzz-bench.conf
COPY docker/nginx.conf.tpl /etc/nginx/nginx.conf.tpl
COPY docker/entrypoint.sh /entrypoint.sh
RUN sed -i 's/\r$//' /entrypoint.sh \
    && chown -R www-data:www-data /var/lib/nginx /var/log/nginx /run/nginx

ENV PHP_JIT=tracing \
    PHP_WORKERS=8 \
    NGINX_WORKERS=1

# Semua proses (nginx + fpm) berjalan sebagai www-data, bukan root.
USER www-data
EXPOSE 8080
# SIGQUIT = graceful shutdown untuk php-fpm (SIGTERM = langsung mati).
STOPSIGNAL SIGQUIT
ENTRYPOINT ["/bin/sh", "/entrypoint.sh"]
