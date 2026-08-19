package v1alpha1

import (
	issuerv1alpha1 "github.com/cert-manager/issuer-lib/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type==\"Ready\")].status"
// +kubebuilder:printcolumn:name="Reason",type="string",JSONPath=".status.conditions[?(@.type==\"Ready\")].reason"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// AkeylessIssuer names an Akeyless PKI issuer that can be referenced by cert-manager Certificates.
type AkeylessIssuer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AkeylessIssuerSpec       `json:"spec,omitempty"`
	Status issuerv1alpha1.IssuerStatus `json:"status,omitempty"`
}

func (a *AkeylessIssuer) GetConditions() []metav1.Condition {
	return a.Status.Conditions
}

func (a *AkeylessIssuer) GetIssuerTypeIdentifier() string {
	return "akeylessissuers.certmanager.akeyless.io"
}

var _ issuerv1alpha1.Issuer = &AkeylessIssuer{}

// +kubebuilder:object:root=true

// AkeylessIssuerList contains a list of AkeylessIssuer resources.
type AkeylessIssuerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AkeylessIssuer `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AkeylessIssuer{}, &AkeylessIssuerList{})
}
