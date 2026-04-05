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
}
