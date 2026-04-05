#!/usr/bin/env bash
# Approve the most recent CertificateRequest in a namespace (local testing).
# Requires: cmctl (https://cert-manager.io/docs/reference/cmctl/)
set -euo pipefail
NS="${NS:-akeyless-cert-test}"
NAME="${1:-$(kubectl get certificaterequest.cert-manager.io -n "$NS" -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true)}"
if [[ -z "$NAME" ]]; then
  echo "No CertificateRequest in namespace $NS" >&2
  exit 1
fi
echo "Approving CertificateRequest $NS/$NAME"
exec cmctl approve -n "$NS" "$NAME"
