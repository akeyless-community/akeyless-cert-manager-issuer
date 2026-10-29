# Akeyless ACME server and cert-manager

## Does this repository implement ACME?

**No.** [akeyless-cert-manager-issuer](https://github.com/akeyless-community/akeyless-cert-manager-issuer) is a **native external issuer**: it calls Akeyless **`/auth`** and **`/get-pki-certificate`** and uses custom CRDs (`AkeylessIssuer`, `AkeylessClusterIssuer`).

[Akeyless ACME Server](https://docs.akeyless.io/docs/acme-server) is a **different integration path**. The gateway exposes a standard **RFC 8555** ACME directory (with **External Account Binding**). For that, you use cert-manager’s **built-in** `Issuer` / `ClusterIssuer` with **`spec.acme`** — you do **not** install or run this external issuer controller.

| Goal | Use |
|------|-----|
| CSR signing via Akeyless API; access key / K8s / cloud IAM auth; `AkeylessIssuer` CRD | **This repo** — [ENVIRONMENT-RECHECK.md](ENVIRONMENT-RECHECK.md) |
| ACME protocol against an Akeyless-hosted CA (`--enable-acme` on a PKI Cert Issuer) | **cert-manager in-tree ACME issuer** — examples below |
| Legacy HVP / Vault-compatible API | cert-manager **`spec.vault`** — [Akeyless cert-manager (HVP)](https://docs.akeyless.io/docs/kubernetes-cert-manager) |

## Prerequisites (Akeyless side)

1. Create a **PKI Cert Issuer** with ACME enabled (`--enable-acme true`) and **allowed domains** set. See [ACME Server](https://docs.akeyless.io/docs/acme-server).
2. Note the **ACME directory URL**:  
   `https://<Your-Akeyless-GW-URL>:<port>/acme/<issuer-display-id>/directory`  
   Get `<issuer-display-id>` from the console or:  
   `akeyless describe-item --name <pki-issuer-path> | jq -r '.display_id'`
3. Generate a one-time **EAB** token:  
   `akeyless generate-acme-eab --cert-issuer-name <pki-issuer-path>`  
   Save `kid` and `mac_key` (store `mac_key` in a Kubernetes `Secret`).

## cert-manager configuration

Use **`cert-manager.io/v1`** `ClusterIssuer` or `Issuer` with **`spec.acme`**. Example manifests: [`examples/acme/`](../examples/acme/).

Important fields:

- **`spec.acme.server`** — full Akeyless ACME directory URL (above).
- **`spec.acme.externalAccountBinding`** — `keyID` = EAB `kid`; `keySecretRef` = Secret containing the EAB HMAC key (`mac_key`).
- **`spec.acme.privateKeySecretRef`** — cert-manager stores the **ACME account** key here (not the EAB secret).
- **`spec.acme.solvers`** — challenge solvers (HTTP-01 / DNS-01) per [cert-manager ACME docs](https://cert-manager.io/docs/configuration/acme/). Configure according to how clients reach your cluster and how domain validation must work for your allowed domains.

If the gateway uses a **private TLS CA**, set **`spec.acme.caBundle`** (cert-manager ≥ 1.11) with the PEM CA that signed the gateway certificate.

### Example flow

```bash
# 1) EAB secret (replace values; do not commit real keys)
kubectl create secret generic akeyless-acme-eab \
  --namespace cert-manager \
  --from-literal=secret='<mac_key from generate-acme-eab>'

# 2) Edit examples/acme/clusterissuer.example.yaml (server URL, keyID, email), then:
kubectl apply -f examples/acme/clusterissuer.example.yaml

# 3) Certificate must use a DNS name in the PKI issuer allowed-domains list
kubectl apply -f examples/acme/certificate.example.yaml
```

`Certificate` **`issuerRef`** must point at the **cert-manager ACME** issuer (`kind: ClusterIssuer`, `name: akeyless-acme`), **not** `AkeylessIssuer`.

## Approval

cert-manager’s **ACME issuer** handles orders and challenges inside cert-manager. You do **not** use `AkeylessIssuer` or approve `CertificateRequest` objects for the external Akeyless signer in this repo (unless you also use that issuer in parallel for other certs).

## When to use both

You can run **both** patterns in one cluster:

- **ACME** `ClusterIssuer` → public or org-internal names via Akeyless ACME + EAB.
- **AkeylessIssuer** (this controller) → direct API issuance with richer Akeyless auth methods and without ACME challenge plumbing.

They are complementary, not overlapping implementations.

## Public CA via this controller (not ACME Server)

If you use an Akeyless **PKI Cert Issuer** backed by a public CA (ZeroSSL / Let's Encrypt) **through this external issuer** (`/get-pki-certificate`), enable:

```yaml
spec:
  asyncIssuance: true
  issuanceTimeout: "15m"
```

That covers the case where the gateway starts ACME asynchronously and the first API response is empty/error while the certificate later appears in Akeyless. Prefer ACME Server (above) when you want cert-manager to own the ACME challenge flow end-to-end.
