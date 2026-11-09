# akeyless-cert-manager-issuer

Repository: **[github.com/akeyless-community/akeyless-cert-manager-issuer](https://github.com/akeyless-community/akeyless-cert-manager-issuer)**.

External [**cert-manager**](https://cert-manager.io/) issuer for [**Akeyless**](https://www.akeyless.io/) PKI (Certificate Issuer / Get PKI Certificate API). Implements `AkeylessIssuer` and `AkeylessClusterIssuer` and signs `CertificateRequest` resources using the Akeyless Go SDK.

## Features

- **PKCS#10 CSR** forwarded to Akeyless `/get-pki-certificate` with PEM encoding expected by the gateway.
- **Authentication** to `/auth` (pick one per issuer): access key, Kubernetes, AWS IAM, Azure AD, GCP, Universal Identity, JWT; legacy `credentialsSecretRef` for access key only.
- **issuer-lib** integration: issuer `Check` + `Sign`, approval-aware `CertificateRequest` reconciliation.
- **Public CA / async issuance** (`spec.asyncIssuance`): wait/poll when the gateway returns before the PEM is ready (Let's Encrypt / ZeroSSL DNS can take minutes), and re-fetch an already-issued cert from Akeyless when possible.

## Documentation

- **[Environment checklist & E2E testing](docs/ENVIRONMENT-RECHECK.md)** — Phase 0 (Akeyless PKI path + DNS alignment), RBAC, controller flags, public vs internal CA tracks, auth examples.
- **[Akeyless ACME server + cert-manager](docs/ACME.md)** — use **cert-manager’s built-in ACME issuer** (not this controller); EAB and example manifests under `examples/acme/`.

## Native API vs ACME

This controller implements the **Akeyless REST API** (`/get-pki-certificate`). It does **not** speak ACME.

If your PKI Cert Issuer has **[ACME Server](https://docs.akeyless.io/docs/acme-server)** enabled on the gateway, configure a standard cert-manager **`ClusterIssuer` with `spec.acme`** pointing at  
`https://<gateway>/acme/<issuer-display-id>/directory` and **External Account Binding** — see [docs/ACME.md](docs/ACME.md).

For **public CA via this controller** (ZeroSSL / Let's Encrypt-backed PKI issuers), set `spec.asyncIssuance: true` (and preferably `issuanceTimeout: "15m"`). Ensure controller `--max-retry-duration` is at least that long (default is **15m**). Rapid recreate of `Certificate` / `CertificateRequest` while ACME is running can race orders.

## Requirements

- Kubernetes cluster with **[cert-manager](https://cert-manager.io/docs/installation/)** installed.
- **Akeyless** API reachable from the controller: Akeyless SaaS (`https://api.akeyless.io`) or your gateway.

## Install

One command installs the CRDs, RBAC and the controller `Deployment` into the `akeyless-cert-manager-system` namespace, using the published image `ghcr.io/akeyless-community/akeyless-cert-manager-issuer:v0.1.2` (linux/amd64):

```bash
kubectl apply -f https://raw.githubusercontent.com/akeyless-community/akeyless-cert-manager-issuer/v0.1.2/deploy/production/install.yaml
```

To upgrade, re-apply the manifest of the new release, or switch only the image:

```bash
kubectl -n akeyless-cert-manager-system set image deploy/akeyless-cert-manager-issuer \
  controller=ghcr.io/akeyless-community/akeyless-cert-manager-issuer:v0.1.2
```

All releases and their images are listed under [Releases](https://github.com/akeyless-community/akeyless-cert-manager-issuer/releases).

## Usage

1. **Credentials and issuer**: start from [`deploy/production/samples/issuer-accesskey.example.yaml`](deploy/production/samples/issuer-accesskey.example.yaml).
   - `apiUrl`: `https://api.akeyless.io` for Akeyless SaaS, or `https://<gateway-host>:8000/api/v2` for a gateway.
   - `certIssuerName`: the full path of your Akeyless PKI Certificate Issuer, e.g. `/Labs/my-internal-ca`.
   - An `AkeylessClusterIssuer` reads its Secret from `akeyless-cert-manager-system`; an `AkeylessIssuer` reads it from its own namespace.
   - Other auth methods (Kubernetes, AWS IAM, Azure AD, GCP, Universal Identity, JWT) are described in [docs/ENVIRONMENT-RECHECK.md](docs/ENVIRONMENT-RECHECK.md#authentication-methods).
2. **Certificate**: create a cert-manager `Certificate` that points at the issuer:

   ```yaml
   issuerRef:
     group: certmanager.akeyless.io
     kind: AkeylessClusterIssuer   # or AkeylessIssuer
     name: akeyless-pki
   ```

3. **Approval**: cert-manager does not approve `CertificateRequest`s for external issuers by default, so the `Certificate` stays pending until its request is approved. Either approve each request manually (`cmctl approve -n <namespace> <certificaterequest-name>`), or allow cert-manager to approve requests for Akeyless issuers automatically:

   ```yaml
   apiVersion: rbac.authorization.k8s.io/v1
   kind: ClusterRole
   metadata:
     name: cert-manager-approve-akeyless
   rules:
     - apiGroups: ["cert-manager.io"]
       resources: ["signers"]
       verbs: ["approve"]
       resourceNames:
         - "akeylessissuers.certmanager.akeyless.io/*"
         - "akeylessclusterissuers.certmanager.akeyless.io/*"
   ---
   apiVersion: rbac.authorization.k8s.io/v1
   kind: ClusterRoleBinding
   metadata:
     name: cert-manager-approve-akeyless
   roleRef:
     apiGroup: rbac.authorization.k8s.io
     kind: ClusterRole
     name: cert-manager-approve-akeyless
   subjects:
     - kind: ServiceAccount
       name: cert-manager        # cert-manager controller service account
       namespace: cert-manager   # namespace cert-manager is installed in
   ```

   If you run [approver-policy](https://cert-manager.io/docs/policy/approval/approver-policy/), configure a `CertificateRequestPolicy` for these issuers instead.

For a full end-to-end walkthrough (PKI issuer setup, DNS names, troubleshooting), see [docs/ENVIRONMENT-RECHECK.md](docs/ENVIRONMENT-RECHECK.md).

Production assets included in this repo:

- `Dockerfile` — multi-stage static build
- `deploy/crds/` — strict CRDs generated by `controller-gen`
- `deploy/production/base/` — namespace, RBAC, service account, deployment, metrics service
- `deploy/production/install.yaml` — single-file install manifest
- `deploy/production/samples/issuer-accesskey.example.yaml` — sample `AkeylessClusterIssuer`

## Quick start (development)

```bash
kubectl apply -f deploy/crds/   # CRDs only; the production install.yaml already includes them
kubectl apply -f examples/local-test/00-namespace.yaml
kubectl apply -f examples/local-test/01-rbac.yaml
# Create Secret + AkeylessIssuer from examples (see docs/ENVIRONMENT-RECHECK.md)
cd /path/to/akeyless-cert-manager-issuer
go run -buildvcs=false ./cmd/controller/ --leader-elect=false
```

The local `go run` flow is meant for development and debugging.
For shared clusters, use the production manifests above: the controller runs as a replicated Kubernetes `Deployment` with **leader election enabled**, hardened container security settings, and generated CRDs.

## Build

```bash
go build -buildvcs=false -o bin/controller ./cmd/controller/
```

## Build your own image

Only needed if you change the code or publish a release; otherwise use the published image above. The binary is built for linux/amd64, so pass `--platform linux/amd64` (required on Apple Silicon so the image platform matches the binary):

```bash
docker build --platform linux/amd64 -t ghcr.io/akeyless-community/akeyless-cert-manager-issuer:v0.1.2 --build-arg VERSION=v0.1.2 .
docker push ghcr.io/akeyless-community/akeyless-cert-manager-issuer:v0.1.2
```

For a custom image, set it in `deploy/production/base/deployment.yaml` or `deploy/production/install.yaml` before applying.

## License

[Apache License 2.0](LICENSE). See [NOTICE](NOTICE) for copyright.

## Contributing

Issues and PRs are welcome. For large changes, open an issue first to agree on approach.
