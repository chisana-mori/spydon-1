#!/usr/bin/env bash
set -euo pipefail

# init.sh - generate TLS keystore and create CAS static auth properties
# Creates:
#  - ../cas/cas.key
#  - ../cas/cas.crt
#  - ../cas/thekeystore (PKCS12 or JKS after conversion)
#  - ../cas/config/cas.properties (with static users)

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
CAS_DIR="$ROOT_DIR/cas"
CONFIG_DIR="$CAS_DIR/config"
KEYFILE="$CAS_DIR/cas.key"
CRTFILE="$CAS_DIR/cas.crt"
P12FILE="$CAS_DIR/thekeystore"
STOREPASS="changeit"

mkdir -p "$CONFIG_DIR"
mkdir -p "$CAS_DIR/runtime/logs"

echo "[init] CAS dir: $CAS_DIR"

if [ ! -f "$KEYFILE" ] || [ ! -f "$CRTFILE" ]; then
  echo "[init] Generating self-signed key and cert"
  openssl req -x509 -nodes -newkey rsa:2048 \
    -keyout "$KEYFILE" -out "$CRTFILE" -days 365 \
    -subj "/CN=localhost/OU=Dev/O=Robusta/L=City/ST=State/C=US"
  chmod 644 "$KEYFILE" "$CRTFILE"
else
  echo "[init] Key/cert already exist, skipping generation"
fi

echo "[init] Creating PKCS#12 keystore at $P12FILE"
openssl pkcs12 -export \
  -in "$CRTFILE" -inkey "$KEYFILE" \
  -name cas -passout pass:"$STOREPASS" \
  -out "$P12FILE"
chmod 644 "$P12FILE"

# Attempt to convert PKCS12 -> JKS using OpenJDK in Docker if available
if command -v docker >/dev/null 2>&1; then
  echo "[init] Converting PKCS12 to JKS using OpenJDK container"
  docker run --rm -v "$CAS_DIR:/work" -w /work openjdk:17-jdk \
    keytool -importkeystore \
      -srckeystore thekeystore -srcstoretype PKCS12 -srcstorepass "$STOREPASS" \
      -destkeystore thekeystore.jks -deststoretype JKS -deststorepass "$STOREPASS" || true
  if [ -f "$CAS_DIR/thekeystore.jks" ]; then
    mv "$CAS_DIR/thekeystore.jks" "$P12FILE"
    echo "[init] Replaced $P12FILE with JKS keystore"
  else
    echo "[init] JKS conversion not completed, leaving PKCS12 as $P12FILE"
  fi
else
  echo "[init] docker not available, skipping JKS conversion (PKCS12 will be used)"
fi

echo "[init] Writing CAS static auth properties to $CONFIG_DIR/cas.properties"
cat > "$CONFIG_DIR/cas.properties" <<EOF
cas.authn.accept.users=yzg::admin,hny::admin,admin::admin
cas.authn.accept.enabled=true
EOF
chmod 644 "$CONFIG_DIR/cas.properties"

echo "[init] Done. Keystore: $P12FILE, properties: $CONFIG_DIR/cas.properties"
echo "[init] Start CAS with: docker-compose -f infrastructure/docker/docker-compose.cas.yml up -d --remove-orphans"
