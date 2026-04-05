package v1alpha1

import corev1 "k8s.io/api/core/v1"

// AkeylessAuth selects how the controller authenticates to Akeyless (/auth).
// Exactly one nested field must be non-nil (unless using legacy credentialsSecretRef on the issuer spec).
type AkeylessAuth struct {
	AccessKey         *AkeylessAuthAccessKey         `json:"accessKey,omitempty"`
	Kubernetes        *AkeylessAuthKubernetes        `json:"kubernetes,omitempty"`
	AwsIam            *AkeylessAuthAwsIam            `json:"awsIam,omitempty"`
	AzureAD           *AkeylessAuthAzureAD           `json:"azureAd,omitempty"`
	Gcp               *AkeylessAuthGcp               `json:"gcp,omitempty"`
	UniversalIdentity *AkeylessAuthUniversalIdentity `json:"universalIdentity,omitempty"`
	Jwt               *AkeylessAuthJwt               `json:"jwt,omitempty"`
}

// AkeylessAuthAccessKey uses access-type access_key (Secret: access-id, access-key).
type AkeylessAuthAccessKey struct {
	SecretRef corev1.LocalObjectReference `json:"secretRef"`
}

// AkeylessAuthKubernetes uses access-type k8s.
type AkeylessAuthKubernetes struct {
	// AccessID is the Akeyless access ID bound to Kubernetes auth (when required by your tenant).
	// +optional
	AccessID string `json:"accessId,omitempty"`
	// K8sAuthConfigName is the Kubernetes auth configuration name in Akeyless / the gateway.
	K8sAuthConfigName string `json:"k8sAuthConfigName"`
	// GatewayURL is sent as gateway-url to /auth (K8s/OAuth2). Often the gateway HTTPS URL without /v2.
	// +optional
	GatewayURL string `json:"gatewayUrl,omitempty"`
	// ServiceAccountTokenSecretRef references a Secret whose data holds the bearer token (bound or legacy SA token).
	ServiceAccountTokenSecretRef corev1.LocalObjectReference `json:"serviceAccountTokenSecretRef"`
	// TokenSecretKey defaults to "token".
	// +optional
	TokenSecretKey string `json:"tokenSecretKey,omitempty"`
}

// AkeylessAuthAwsIam uses access-type aws_iam. Cloud ID is taken from spec/secret, or resolved via akeyless-go-cloud-id on the controller node.
type AkeylessAuthAwsIam struct {
	AccessID string `json:"accessId"`
	// CloudID optional; if empty, resolved from instance/workload metadata when possible.
	// +optional
	CloudID string `json:"cloudId,omitempty"`
	// CloudIDSecretRef references a Secret key holding the cloud identity string.
	// +optional
	CloudIDSecretRef *corev1.LocalObjectReference `json:"cloudIdSecretRef,omitempty"`
	// CloudIDSecretKey defaults to "cloud-id".
	// +optional
	CloudIDSecretKey string `json:"cloudIdSecretKey,omitempty"`
}

// AkeylessAuthAzureAD uses access-type azure_ad.
type AkeylessAuthAzureAD struct {
	AccessID string `json:"accessId"`
	// +optional
	CloudID string `json:"cloudId,omitempty"`
	// +optional
	CloudIDSecretRef *corev1.LocalObjectReference `json:"cloudIdSecretRef,omitempty"`
	// +optional
	CloudIDSecretKey string `json:"cloudIdSecretKey,omitempty"`
}

// AkeylessAuthGcp uses access-type gcp.
type AkeylessAuthGcp struct {
	AccessID string `json:"accessId"`
	// +optional
	GcpAudience string `json:"gcpAudience,omitempty"`
	// +optional
	CloudID string `json:"cloudId,omitempty"`
	// +optional
	CloudIDSecretRef *corev1.LocalObjectReference `json:"cloudIdSecretRef,omitempty"`
	// +optional
	CloudIDSecretKey string `json:"cloudIdSecretKey,omitempty"`
}

// AkeylessAuthUniversalIdentity uses access-type universal_identity (Secret key uid-token by default).
type AkeylessAuthUniversalIdentity struct {
	SecretRef corev1.LocalObjectReference `json:"secretRef"`
	// UidTokenSecretKey defaults to "uid-token".
	// +optional
	UidTokenSecretKey string `json:"uidTokenSecretKey,omitempty"`
}

// AkeylessAuthJwt uses access-type jwt (OIDC tokens are supplied the same way per Akeyless API).
type AkeylessAuthJwt struct {
	// +optional
	AccessID string `json:"accessId,omitempty"`
	SecretRef corev1.LocalObjectReference `json:"secretRef"`
	// JwtSecretKey defaults to "jwt".
	// +optional
	JwtSecretKey string `json:"jwtSecretKey,omitempty"`
}
