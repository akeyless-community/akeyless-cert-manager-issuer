# akeyless-cert-manager-issuer

Repository: **[github.com/akeyless-community/akeyless-cert-manager-issuer](https://github.com/akeyless-community/akeyless-cert-manager-issuer)**.

If you truly need the repo name `akeyless-cermanager`, create that name on GitHub instead, then set `go.mod`’s first line to `module github.com/akeyless-community/akeyless-cermanager` and replace all import paths accordingly — the code here assumes **`akeyless-cert-manager-issuer`** (matches the project and avoids the typo “cermanager”).

External [**cert-manager**](https://cert-manager.io/) issuer for [**Akeyless**](https://www.akeyless.io/) PKI (Certificate Issuer / Get PKI Certificate API). Implements `AkeylessIssuer` and `AkeylessClusterIssuer` and signs `CertificateRequest` resources using the Akeyless Go SDK.

## Features

- **PKCS#10 CSR** forwarded to Akeyless `/get-pki-certificate` with PEM encoding expected by the gateway.
- **Authentication** to `/auth` (pick one per issuer): access key, Kubernetes, AWS IAM, Azure AD, GCP, Universal Identity, JWT; legacy `credentialsSecretRef` for access key only.
- **issuer-lib** integration: issuer `Check` + `Sign`, approval-aware `CertificateRequest` reconciliation.

## Documentation

- **[Environment checklist & E2E testing](docs/ENVIRONMENT-RECHECK.md)** — Phase 0 (Akeyless PKI path + DNS alignment), RBAC, controller flags, public vs internal CA tracks, auth examples.

## Requirements

- Kubernetes cluster with **cert-manager** installed.
- **Akeyless** gateway reachable from the controller (in-cluster `Deployment` or local `go run` with `KUBECONFIG`).
- This repository’s **CRDs** applied (`deploy/crds/`).

## Quick start (development)

```bash
kubectl apply -f deploy/crds/
kubectl apply -f examples/local-test/00-namespace.yaml
kubectl apply -f examples/local-test/01-rbac.yaml
# Create Secret + AkeylessIssuer from examples (see docs/ENVIRONMENT-RECHECK.md)
cd /path/to/akeyless-cert-manager-issuer
go run -buildvcs=false ./cmd/controller/ --leader-elect=false
```

Production deployments should run a **versioned container image** with **leader election** enabled and a restricted CRD schema (regenerate with kubebuilder/controller-gen when you promote beyond dev CRDs).

## Build

```bash
go build -buildvcs=false -o bin/controller ./cmd/controller/
```

## License

[Apache License 2.0](LICENSE). See [NOTICE](NOTICE) for copyright.

## Listing on cert-manager.io

The cert-manager project maintains a **public table of external issuers** on the documentation site:

- **Page:** [Issuers (external)](https://cert-manager.io/docs/configuration/external/)

**To add this issuer to that list**

1. **Publish this repository** (GitHub or elsewhere) with a clear **README**, **license**, and **releases** (tags or changelog) so users can install a known version.
2. Open a **Pull Request** against the cert-manager **website** repo that edits the issuers table:
   - **Repository:** [github.com/cert-manager/website](https://github.com/cert-manager/website)
   - **Typical file:** `content/docs/configuration/external.md` (path may change; search the repo for `aws-privateca-issuer` or `step-issuer` to find the markdown source for the table).
3. **Add a row** following the existing columns, for example:
   - **Tier** — start with **🥈 (Maintained)** if you have releases within the last 12 months and ongoing maintenance; **🥇** requires a full **production end-to-end tutorial** (install cert-manager version + install issuer + configure CRs + issue a `Certificate`) as described on that page.
   - **Controller** — short name, e.g. `akeyless-cert-manager-issuer`.
   - **Docs** — link to this repo’s README or to Akeyless docs for install/configuration.
   - **Issuer** — link to Akeyless PKI / certificate issuance product docs.
   - Other links (cert-manager version, releases, open source) per the table’s conventions.

4. In the PR description, note that the issuer **honours CertificateRequest approval** (same model as other issuer-lib-based issuers).

Maintainers may adjust tier criteria over time; see the **Issuer Tier system** section on the same documentation page.

## Push this tree to GitHub (`akeyless-community`)

Git is already initialized with `main` and an initial commit in this directory.

1. In the [Akeyless Community](https://github.com/orgs/akeyless-community) org, click **New repository**.
2. Name it **`akeyless-cert-manager-issuer`**, leave it **empty** (no README, no `.gitignore`, no license — this tree already has them).
3. Add the remote and push:

```bash
cd /path/to/akeyless-cert-manager-issuer
git remote add origin https://github.com/akeyless-community/akeyless-cert-manager-issuer.git
git push -u origin main
```

Use SSH if you prefer: `git@github.com:akeyless-community/akeyless-cert-manager-issuer.git`.

## Contributing

Issues and PRs are welcome. For large changes, open an issue first to agree on approach.
