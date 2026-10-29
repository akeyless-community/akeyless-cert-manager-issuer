package v1alpha1

import corev1 "k8s.io/api/core/v1"

// AkeylessIssuerSpec defines connection settings for the Akeyless PKI Certificate Issuer.
type AkeylessIssuerSpec struct {
	// APIURL is the base URL of the Akeyless REST API (v2), e.g. https://gateway.example.com:8080/v2
	// +kubebuilder:validation:MinLength=1
	APIURL string `json:"apiUrl"`

	// CertIssuerName is the full Akeyless name of the PKI Certificate Issuer (e.g. /dev/my-pki-issuer).
	// +kubebuilder:validation:MinLength=1
	CertIssuerName string `json:"certIssuerName"`

	// Auth configures how to call Akeyless /auth. Exactly one method must be set inside auth.
	// If auth is nil, credentialsSecretRef must be set (legacy API-key-only behavior).
	// +optional
	Auth *AkeylessAuth `json:"auth,omitempty"`

	// CredentialsSecretRef references a Secret with keys access-id and access-key (access_key auth).
	// Deprecated: prefer auth.accessKey.secretRef.
	// +optional
	CredentialsSecretRef *corev1.LocalObjectReference `json:"credentialsSecretRef,omitempty"`

	// AsyncIssuance enables wait/poll behavior for public CA issuers (Let's Encrypt, ZeroSSL, etc.)
	// where /get-pki-certificate may return before the certificate PEM is ready. When true, Sign
	// holds until the cert can be fetched or issuanceTimeout elapses, instead of failing immediately.
	// +optional
	AsyncIssuance bool `json:"asyncIssuance,omitempty"`

	// RequestTimeout is the per-HTTP-call timeout for Akeyless API requests (Go duration, e.g. "2m").
	// Defaults to 2m.
	// +optional
	// +kubebuilder:validation:Pattern=`^([0-9]+(\.[0-9]+)?(ns|us|µs|ms|s|m|h))+$`
	RequestTimeout string `json:"requestTimeout,omitempty"`

	// IssuanceTimeout is the total wait budget for one Sign attempt including polls (e.g. "15m").
	// Defaults to 2m, or 15m when asyncIssuance is true. Should be <= controller --max-retry-duration.
	// +optional
	// +kubebuilder:validation:Pattern=`^([0-9]+(\.[0-9]+)?(ns|us|µs|ms|s|m|h))+$`
	IssuanceTimeout string `json:"issuanceTimeout,omitempty"`
}
