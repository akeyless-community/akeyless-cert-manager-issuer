# Akeyless cert-manager issuer — environment checklist and testing

Use this document when validating **akeyless-cert-manager-issuer** on a new cluster or after upgrades. It mirrors what the controller actually implements (see `api/v1alpha1`, `controller/auth.go`, `examples/local-test/`).

## Prerequisites

- **Kubernetes** with working `kubectl` (or equivalent) and a default StorageClass if you rely on PVCs elsewhere (not required for this issuer).
- **cert-manager** installed and healthy (`cert-manager` pods running).
- **cmctl** (recommended): [cert-manager cmctl](https://cert-manager.io/docs/reference/cmctl/) — used to approve `CertificateRequest` objects when you do not use an approver.
- **Akeyless** gateway reachable from wherever the issuer controller runs (cluster pods or your laptop).
- A **PKI Certificate Issuer** configured in Akeyless whose name you will put in `spec.certIssuerName`.
- **CRDs** for this project applied (`deploy/crds/*.yaml`).

Confirm your **API base URL**: the Go client posts to paths such as `/auth` and `/get-pki-certificate` under the configured server URL. Use the **v2 API base** your gateway expects (often `https://<gateway-host>:<port>/v2` — align with [Akeyless API](https://docs.akeyless.io/) for your deployment).

---

## Phase 0: Akeyless PKI issuer and DNS names

The **PKI Certificate Issuer** in Akeyless must be able to sign the **exact DNS names** (and CN) that cert-manager puts in the CSR. The repo examples use two fixed tracks so you never mix a ZeroSSL issuer with `.svc.cluster.local` names (which will fail).

Pick **one** track and use the **paired** manifests:

| Track | Use case | Kubernetes `AkeylessIssuer` **name** (`metadata.name`) | Akeyless PKI path (`spec.certIssuerName`) | cert-manager `Certificate` manifest | YAML for issuer |
|-------|----------|--------------------------------------------------------|--------------------------------------------|---------------------------------------|-----------------|
| **A — Internal / private CA** | Dev clusters, in-cluster TLS | **`akeyless-pki`** | e.g. **`/Labs/k8s-e2e-internal-ca`** | `examples/local-test/04-certificate.yaml` | `examples/local-test/03-akeylessissuer.internal.example.yaml` |
| **B — Public CA (ZeroSSL)** | Real hostname on the internet | **`akeyless-pki-public`** | e.g. **`/public-ca/zerossl/zerossl`** | `examples/local-test/04-certificate.public-dns.example.yaml` | `examples/local-test/03-akeylessissuer.public.example.yaml` |

**Important:** `kubectl describe akeylessissuer …` takes the **Kubernetes resource name** (`akeyless-pki`), **not** the last segment of the Akeyless path. For example, **`k8s-e2e-internal-ca`** is only part of `/Labs/k8s-e2e-internal-ca` inside `spec.certIssuerName` — it is **not** the CR name unless you set `metadata.name: k8s-e2e-internal-ca` yourself.

**Canonical DNS names (Track A)** — already set in `04-certificate.yaml`:

- **Namespace:** `akeyless-cert-test`
- **CN and SAN:** `demo-tls.akeyless-cert-test.svc.cluster.local`  
  (Kubernetes DNS form: a Service named `demo-tls` in that namespace would match this name.)

If you change the namespace, update **both** the `Certificate` namespace **and** the FQDN so it stays `<name>.<namespace>.svc.cluster.local`.

**Canonical DNS names (Track B)** — you edit the copy of the example:

- Replace **`YOUR_PUBLIC_FQDN`** with one hostname you are allowed to issue for (e.g. `k8s-e2e-api.prod.example.com`).
- That hostname must be **approved** by your Public CA flow (DNS ownership / ACME / ZeroSSL rules), not a made-up TLD like `.test`.

### Track A — steps in Akeyless (internal / private PKI)

1. In **Akeyless** (Console or API), create a **PKI Certificate Issuer** intended for **internal** issuance (subordinate CA, internal CA, or equivalent — **not** the ZeroSSL public-CA integration if you need arbitrary in-cluster names).  
   See: [Akeyless PKI / certificate issuer](https://docs.akeyless.io/docs/certificate-issuance) and related topics for your deployment type.

2. Set the issuer’s **full path** to the value you will use in Kubernetes, for example:  
   **`/Labs/k8s-e2e-internal-ca`**  
   (You may use another path; if you do, change `spec.certIssuerName` in `03-akeylessissuer.internal.example.yaml` to match exactly.)

3. **Allowed / permitted DNS names** (wording depends on Akeyless version and issuer type):
   - If the issuer lets you restrict **allowed domains** or **SAN patterns**, include at least:
     - `demo-tls.akeyless-cert-test.svc.cluster.local`
     - and/or a wildcard that covers it, e.g. `*.akeyless-cert-test.svc.cluster.local` or `*.svc.cluster.local` (the latter is broad — use only in lab).
   - If your **internal CA has no SAN allow-list**, you can skip this; the CSR will still carry the names below.

4. Ensure the **identity** you will use for `/auth` (access key, K8s auth, cloud IAM, etc.) has permission to call **Get PKI Certificate** on that issuer path.

5. In Kubernetes, set `spec.certIssuerName` on `AkeylessIssuer` to the **same** path string (no trailing slash).

### Track B — steps in Akeyless (Public CA / ZeroSSL)

1. Configure a **PKI Certificate Issuer** in Akeyless that uses **Public CA** / **ZeroSSL** (or your org’s equivalent), following [Akeyless documentation](https://docs.akeyless.io/).

2. Complete any **ZeroSSL account / domain validation** steps so the CA will issue for your real zone (e.g. `example.com`).

3. Note the issuer’s **full path** (example: **`/public-ca/zerossl/zerossl`**). Put that exact string in `03-akeylessissuer.public.example.yaml` as `spec.certIssuerName` if yours differs.

4. Copy `04-certificate.public-dns.example.yaml` to a working file, replace **`YOUR_PUBLIC_FQDN`** with a hostname under your approved zone (e.g. `demo-k8s-e2e.prod.example.com`). Use the **same** string for `commonName` and `dnsNames` unless you intend multiple SANs.

5. Do **not** use Track B for `*.svc.cluster.local` — public CAs typically reject those (`certificate_unsupported_top_level_domain` or similar).

---

## End-to-end checklist (new environment)

Work through the steps in order. **Complete [Phase 0](#phase-0-akeyless-pki-issuer-and-dns-names)** first so PKI path and certificate names match.

### 1. Install cert-manager

```bash
kubectl get pods -n cert-manager
```

### 2. Apply Akeyless issuer CRDs

From the repository root:

```bash
kubectl apply -f deploy/crds/
```

Verify:

```bash
kubectl get crd akeylessissuers.certmanager.akeyless.io
kubectl get crd akeylessclusterissuers.certmanager.akeyless.io
```

### 3. Namespace (optional but recommended)

```bash
kubectl apply -f examples/local-test/00-namespace.yaml
```

### 4. RBAC for the issuer controller

Apply a `ClusterRole` + `ClusterRoleBinding` (or equivalent) so the controller can:

- Read/watch/patch `certificaterequests` and `certificaterequests/status` (`cert-manager.io`)
- Read/watch/patch `certificatesigningrequests` and `.../status` (`certificates.k8s.io`) if you use CSR integration
- `sign` on `signers` with resource names `akeylessissuers.certmanager.akeyless.io/*` and `akeylessclusterissuers.certmanager.akeyless.io/*`
- Read/watch/patch `akeylessissuers`, `akeylessclusterissuers` and their `status` (`certmanager.akeyless.io`)
- **Get/list/watch** `secrets` (to read credentials referenced by the issuer)
- **coordination.k8s.io/leases** if **leader election** is enabled (typical in-cluster)
- **events** if you emit events (issuer-lib does)

Example (local dev–oriented):

```bash
kubectl apply -f examples/local-test/01-rbac.yaml
```

If the controller runs **in-cluster** under a `ServiceAccount`, bind that SA to the role instead of the example namespace.

### 5. Credential Secrets and `AkeylessIssuer`

- Create the Secret(s) required for your chosen auth method (see [Authentication methods](#authentication-methods) below).
- Create `AkeylessIssuer` with `spec.apiUrl`, `spec.certIssuerName`, and **exactly one** auth configuration.

**Track A (internal PKI + `04-certificate.yaml`):**

```bash
# 1) Copy and fill access-id / access-key
cp examples/local-test/02-secret-api-key.example.yaml /tmp/02-secret.yaml
# edit /tmp/02-secret.yaml, then:
kubectl apply -f /tmp/02-secret.yaml

# 2) Copy issuer example; set apiUrl + certIssuerName to match Phase 0
cp examples/local-test/03-akeylessissuer.internal.example.yaml /tmp/03-issuer.yaml
# edit /tmp/03-issuer.yaml, then:
kubectl apply -f /tmp/03-issuer.yaml
```

**Track B (public DNS + ZeroSSL-style issuer):**

```bash
cp examples/local-test/02-secret-api-key.example.yaml /tmp/02-secret.yaml   # edit
kubectl apply -f /tmp/02-secret.yaml
cp examples/local-test/03-akeylessissuer.public.example.yaml /tmp/03-issuer-public.yaml
# edit apiUrl, certIssuerName (must match your Akeyless public PKI path)
kubectl apply -f /tmp/03-issuer-public.yaml
```

### 6. Run the issuer controller

**In-cluster (recommended for production):** deploy a `Deployment` that runs the controller image with your SA; enable leader election (default). Ensure `Lease` RBAC exists in the **leader election namespace** (usually the pod namespace).

**Local (kubeconfig on your machine):**

```bash
cd akeyless-cert-manager-issuer
go run -buildvcs=false ./cmd/controller/ --leader-elect=false
```

If you keep `--leader-elect=true` outside the cluster, you **must** set `--leader-election-namespace=<ns>` and grant **leases** in that namespace.

Useful flags (see `cmd/controller/main.go`):

| Flag | Purpose |
|------|--------|
| `--leader-elect=false` | Single local process; no `Lease` objects |
| `--leader-election-namespace` | Required for leader election when not in-cluster |
| `--cluster-resource-namespace` | Namespace where Secrets for **`AkeylessClusterIssuer`** live (required for cluster issuers when not inferred from the pod) |
| `--metrics-bind-address` | Default `:8080` |
| `--health-probe-bind-address` | Default `:8081` (local only; unrelated to Akeyless gateway port) |
| `--max-retry-duration` | How long signing retries before marking the `CertificateRequest` failed |

Build note: if the module is not in a git repo, use `-buildvcs=false` or `GOFLAGS=-buildvcs=false`.

### 7. Verify issuer health (`Check`)

Use the **`metadata.name`** of the `AkeylessIssuer` (examples: **`akeyless-pki`** for Track A, **`akeyless-pki-public`** for Track B):

```bash
kubectl get akeylessissuer -n akeyless-cert-test
kubectl describe akeylessissuer -n akeyless-cert-test akeyless-pki
```

You want **`Ready=True`** and a message indicating the issuer check succeeded (Akeyless `/auth` works with your credentials). If `NotFound`, apply the issuer YAML first or fix the name with `kubectl get akeylessissuer -n akeyless-cert-test -o name`.

### 8. Create a `Certificate`

Names must match [Phase 0](#phase-0-akeyless-pki-issuer-and-dns-names).

**Track A:**

```bash
# Already aligned: CN + SAN = demo-tls.akeyless-cert-test.svc.cluster.local
kubectl apply -f examples/local-test/04-certificate.yaml
```

**Track B:**

```bash
cp examples/local-test/04-certificate.public-dns.example.yaml /tmp/04-cert-public.yaml
# Replace YOUR_PUBLIC_FQDN with your real hostname (e.g. demo-k8s-e2e.prod.example.com)
kubectl apply -f /tmp/04-cert-public.yaml
```

Then approve the **new** `CertificateRequest` name (Track B uses certificate name `demo-tls-public`, so the first request is often `demo-tls-public-1`).

### 9. Approve the `CertificateRequest` (if required)

If `CertificateRequest` does not auto-approve:

```bash
kubectl get certificaterequest -n akeyless-cert-test
cmctl approve -n akeyless-cert-test <certificaterequest-name>
```

Or use `examples/local-test/approve-latest-certificaterequest.sh` (set `NS` if needed).

The argument to `cmctl approve` is the **resource name** (e.g. `demo-tls-1` for Track A, or `demo-tls-public-1` for Track B), not the DNS name in the CSR.

### 10. Confirm certificate issuance

```bash
kubectl describe certificaterequest -n akeyless-cert-test <name>
kubectl get certificate -n akeyless-cert-test
kubectl get secret -n akeyless-cert-test <certificate.spec.secretName>
```

Expect **`Ready=True`** on the `CertificateRequest`, then a populated TLS `Secret`.

### 11. If something failed

- **`CertificateRequest` stuck without `Approved`:** run `cmctl approve` or install/configure an approver (e.g. approver-policy).
- **`Failed` after retries:** read the controller logs and the CR `status`; fix issuer/Akeyless/PKI config, then **renew** the `Certificate` (`cmctl renew ...`) or delete the failed `CertificateRequest` so a new one is created.
- **Leader election errors locally:** `--leader-elect=false` or `--leader-election-namespace=...`.

---

## Authentication methods

The issuer builds an Akeyless **`/auth`** body from `AkeylessIssuer` / `AkeylessClusterIssuer` `spec`. You must set **exactly one** of:

- `spec.auth.accessKey` | `spec.auth.kubernetes` | `spec.auth.awsIam` | `spec.auth.azureAd` | `spec.auth.gcp` | `spec.auth.universalIdentity` | `spec.auth.jwt`
- **Or** legacy `spec.credentialsSecretRef` (access-id + access-key only), if `spec.auth` is omitted.

Secret keys for access keys are **`access-id`** and **`access-key`** (see `controller/signer.go` constants).

### Access key (`access_key`)

**When to use:** simplest tests, CI, or when a static access key is acceptable.

**Secret:**

```yaml
stringData:
  access-id: "<Akeyless access ID>"
  access-key: "<Akeyless access key>"
```

**Issuer snippet:**

```yaml
spec:
  apiUrl: "https://your-gateway:8080/v2"
  certIssuerName: "/path/to/pki-issuer"
  auth:
    accessKey:
      secretRef:
        name: akeyless-api-key
```

Legacy equivalent: `credentialsSecretRef.name: akeyless-api-key` (same secret keys).

---

### Kubernetes (`k8s`)

**When to use:** workloads in the same (or trusted) cluster; Akeyless **Kubernetes auth method** configured with your API server and RBAC.

**What the controller sends:** `access-type: k8s`, `k8s-auth-config-name`, `k8s-service-account-token`, optional `access-id`, optional `gateway-url`.

**You need:**

1. In Akeyless: a **K8s auth config** (`k8sAuthConfigName`) bound to the Kubernetes API and the **ServiceAccount** you will use.
2. A **Secret** in the issuer’s namespace containing the SA **bearer token** (key defaults to `token`).
3. Optional `gatewayUrl` on the auth spec if your `/auth` flow requires it (same host you use for API calls is common; omit trailing slash).

**Issuer snippet:**

```yaml
spec:
  apiUrl: "https://your-gateway:8080/v2"
  certIssuerName: "/path/to/pki-issuer"
  auth:
    kubernetes:
      k8sAuthConfigName: "my-k8s-auth-config"
      serviceAccountTokenSecretRef:
        name: akeyless-k8s-sa-token
      # accessId: "..."           # if your method requires it
      # gatewayUrl: "https://your-gateway:8080"
      # tokenSecretKey: token     # default
```

**Prepare the token Secret (examples):**

- **Kubernetes 1.24+:** long-lived tokens are not auto-created in SA secrets. Typical options:
  - **Projected volume** in a small pod that writes the token into a Secret (via sidecar/init) — or
  - `kubectl create token <sa> --duration=...` and paste into a Secret (good for tests; rotate for production).
- Ensure the **issuer controller** can `get` that Secret (same namespace as `AkeylessIssuer` for namespaced issuers).

**Test flow:** apply the Secret + issuer with `auth.kubernetes`, confirm `AkeylessIssuer` **Ready**, then create a `Certificate` and approve the `CertificateRequest` as usual.

---

### AWS IAM (`aws_iam`)

**When to use:** controller runs on **AWS** (e.g. EKS worker/node or pod with IMDS/IRSA) and Akeyless auth method is **AWS IAM**.

**Required:** `accessId` (Akeyless access ID for IAM auth).

**Cloud ID:** sent as `cloud-id` to Akeyless. Resolution order:

1. `spec.auth.awsIam.cloudId` (inline), or  
2. Secret referenced by `cloudIdSecretRef` (key defaults to `cloud-id`), or  
3. Auto-discovery via **`akeyless-go-cloud-id`** on the machine running the controller (metadata service, etc.).

**Issuer snippet:**

```yaml
spec:
  apiUrl: "https://your-gateway:8080/v2"
  certIssuerName: "/path/to/pki-issuer"
  auth:
    awsIam:
      accessId: "<Akeyless IAM access ID>"
      # cloudId: "..."                    # optional if auto-detected
      # cloudIdSecretRef:
      #   name: aws-cloud-id
      # cloudIdSecretKey: cloud-id       # default
```

**Testing tips:**

- **Controller Pod on EKS:** use IRSA or node role so metadata-based cloud ID matches what Akeyless expects; or put the cloud ID string in a Secret.
- **Controller on a laptop:** metadata lookup usually **fails** — set `cloudId` or `cloudIdSecretRef`, or run the controller inside AWS.

---

### Azure AD (`azure_ad`)

**When to use:** controller runs on **Azure** and Akeyless auth uses **Azure AD**.

Same **cloud ID** pattern as AWS: inline, Secret (`cloud-id` key by default), or auto-discovery when running on Azure.

```yaml
auth:
  azureAd:
    accessId: "<Akeyless Azure AD access ID>"
    # cloudId / cloudIdSecretRef as needed
```

---

### GCP (`gcp`)

**When to use:** controller runs on **GCP** (GKE, GCE) with Akeyless **GCP** auth.

**Required:** `accessId`. Optional **`gcpAudience`** maps to Akeyless `gcp-audience` when your setup requires it.

Cloud ID: same triad as AWS/Azure (inline, Secret, or metadata).

```yaml
auth:
  gcp:
    accessId: "<Akeyless GCP access ID>"
    # gcpAudience: "..."   # optional
    # cloudId / cloudIdSecretRef as needed
```

---

### Universal Identity (`universal_identity`)

**Secret** (default key `uid-token`):

```yaml
auth:
  universalIdentity:
    secretRef:
      name: akeyless-uid-token
    # uidTokenSecretKey: uid-token
```

### JWT (`jwt`)

**Secret** (default key `jwt`):

```yaml
auth:
  jwt:
    secretRef:
      name: akeyless-jwt
    # jwtSecretKey: jwt
    # accessId: "..."   # optional
```

---

## `AkeylessClusterIssuer`

- Same `spec` shape as `AkeylessIssuer`, but cluster-scoped.
- Secrets referenced by the issuer are read from **`--cluster-resource-namespace`** when the controller cannot infer a namespace from the issuer object (required for local/out-of-cluster runs; in-cluster the default may be the pod namespace — see `cmd/controller/main.go`).

---

## Quick reference: objects and flows

| Resource | Role |
|----------|------|
| `AkeylessIssuer` | Namespaced; references Akeyless API + PKI issuer + auth |
| `Certificate` (cert-manager) | Requests TLS cert; creates `CertificateRequest` |
| `CertificateRequest` | Holds CSR; must be **Approved** then signed by this issuer |
| Issuer controller | Calls Akeyless `/auth`, then `/get-pki-certificate` with CSR |

CSR handling: the controller sends `csr-data-base64` as **base64(PEM CSR)** per Akeyless expectations (not raw DER-only).

---

## Suggested order for testing multiple auth methods

1. **Access key** — fastest path to prove gateway, PKI issuer, cert-manager, and controller wiring.  
2. **Kubernetes** — validate SA token Secret + Akeyless K8s auth config.  
3. **Cloud IAM** — deploy controller in the matching cloud (or supply `cloudId` via Secret) and test **AWS → Azure → GCP** as needed for your environments.

After each method change, watch **`AkeylessIssuer` Ready** and a **full certificate issuance** (not only `/auth`).

---

## Repository paths

| Path | Content |
|------|---------|
| `deploy/crds/` | CRD manifests |
| `examples/local-test/` | Namespace, RBAC, secrets; issuer+cert **pairs** for Track A / B (see Phase 0) |
| `api/v1alpha1/auth_types.go` | Auth field definitions |
| `controller/auth.go` | Auth body construction |
| `internal/akeylessapi/` | `/auth` and `/get-pki-certificate` client |
| `README.md` | Overview, build, and how to propose listing on [cert-manager.io](https://cert-manager.io/docs/configuration/external/) |

For Akeyless-side setup (K8s auth method, PKI issuers, public CA constraints), use [Akeyless documentation](https://docs.akeyless.io/).

To **publish the project** (git hygiene, license, listing in the upstream issuer table), follow the **Listing on cert-manager.io** section in the repo [`README.md`](../README.md).
