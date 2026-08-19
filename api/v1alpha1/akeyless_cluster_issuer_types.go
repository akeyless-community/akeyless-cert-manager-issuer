package v1alpha1

import (
	issuerv1alpha1 "github.com/cert-manager/issuer-lib/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type==\"Ready\")].status"
// +kubebuilder:printcolumn:name="Reason",type="string",JSONPath=".status.conditions[?(@.type==\"Ready\")].reason"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// AkeylessClusterIssuer is a cluster-scoped Akeyless PKI issuer.
type AkeylessClusterIssuer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AkeylessIssuerSpec       `json:"spec,omitempty"`
	Status issuerv1alpha1.IssuerStatus `json:"status,omitempty"`
}

func (a *AkeylessClusterIssuer) GetConditions() []metav1.Condition {
	return a.Status.Conditions
}

func (a *AkeylessClusterIssuer) GetIssuerTypeIdentifier() string {
	return "akeylessclusterissuers.certmanager.akeyless.io"
}

var _ issuerv1alpha1.Issuer = &AkeylessClusterIssuer{}

// +kubebuilder:object:root=true

// AkeylessClusterIssuerList contains a list of AkeylessClusterIssuer resources.
type AkeylessClusterIssuerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AkeylessClusterIssuer `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AkeylessClusterIssuer{}, &AkeylessClusterIssuerList{})
}
