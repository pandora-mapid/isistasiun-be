#!/bin/sh
# One-time Let's Encrypt bootstrap. Run on the VPS after:
#   1. DNS for $SERVER_NAME points at this box
#   2. /opt/isistasiun/.env exists
#   3. `docker compose -f docker-compose.deploy.yml --env-file .env up -d` has run once
#
#   ./scripts/init-letsencrypt.sh
#
# nginx cannot start with a 443 block that references a missing cert, so this
# drops in a dummy self-signed cert, starts nginx, then swaps in the real one.
set -e

cd "$(dirname "$0")/.."
. ./.env

DOMAIN="$SERVER_NAME"
CONF="./nginx/certbot/conf"
LIVE="$CONF/live/$DOMAIN"
COMPOSE="docker compose -f docker-compose.deploy.yml --env-file .env"

if [ -f "$LIVE/fullchain.pem" ] && [ ! -f "$LIVE/.dummy" ]; then
  echo "real certificate for $DOMAIN already present — nothing to do"
  exit 0
fi

echo "### dummy certificate for $DOMAIN"
mkdir -p "$LIVE"
$COMPOSE run --rm --entrypoint "sh -c" certbot \
  "openssl req -x509 -nodes -newkey rsa:2048 -days 1 \
     -keyout /etc/letsencrypt/live/$DOMAIN/privkey.pem \
     -out /etc/letsencrypt/live/$DOMAIN/fullchain.pem -subj /CN=localhost"
touch "$LIVE/.dummy"

echo "### starting nginx with the dummy cert"
$COMPOSE up -d --force-recreate nginx

echo "### deleting dummy, requesting real certificate"
$COMPOSE run --rm --entrypoint "sh -c" certbot \
  "rm -rf /etc/letsencrypt/live/$DOMAIN /etc/letsencrypt/archive/$DOMAIN /etc/letsencrypt/renewal/$DOMAIN.conf"
$COMPOSE run --rm --entrypoint "sh -c" certbot \
  "certbot certonly --webroot -w /var/www/certbot -d $DOMAIN \
     --email $CERTBOT_EMAIL --agree-tos --no-eff-email --non-interactive"

echo "### bringing the full stack up"
$COMPOSE up -d
$COMPOSE exec nginx nginx -s reload || true
echo "done — https://$DOMAIN"
