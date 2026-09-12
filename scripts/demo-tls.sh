#!/usr/bin/env bash
# Serves the demo over HTTPS so a phone on the same Wi-Fi can install the driver PWA.
set -euo pipefail
export PATH=/opt/homebrew/bin:$PATH
cd "$(dirname "$0")/.."

command -v mkcert >/dev/null || { echo "mkcert is required: brew install mkcert && mkcert -install" >&2; exit 1; }
LAN=$(ipconfig getifaddr en0 2>/dev/null || hostname -I 2>/dev/null | awk '{print $1}')
mkdir -p certs
if [ ! -f certs/cert.pem ]; then
  mkcert -cert-file certs/cert.pem -key-file certs/key.pem localhost 127.0.0.1 ${LAN:-}
fi
make -s build
if [ ! -f data/demo.sqlite ]; then
  DEEPCUTS_DB_PATH=data/demo.sqlite bin/deepcuts seed demo
fi
cat <<EOF

Phone setup (one time): install the mkcert root CA on the phone so the certificate is trusted.
  Root CA file: $(mkcert -CAROOT)/rootCA.pem  (AirDrop it, then Settings → General → VPN & Device Management → install, and enable full trust under Certificate Trust Settings on iOS)

Driver app: https://${LAN:-localhost}:8443/driver   (add to home screen to install)
Office:     https://${LAN:-localhost}:8443/office

EOF
DEEPCUTS_DB_PATH=data/demo.sqlite DEEPCUTS_ADDR=:8443 DEEPCUTS_TLS_CERT=certs/cert.pem DEEPCUTS_TLS_KEY=certs/key.pem bin/deepcuts serve
