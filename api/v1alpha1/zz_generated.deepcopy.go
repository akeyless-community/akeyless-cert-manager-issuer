package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto copies AkeylessIssuerSpec into out.
func (in *AkeylessIssuerSpec) DeepCopyInto(out *AkeylessIssuerSpec) {
	*out = *in
	if in.Auth != nil {
		in, out := &in.Auth, &out.Auth
		*out = new(AkeylessAuth)
		(*in).DeepCopyInto(*out)
	} else {
		out.Auth = nil
	}
	if in.CredentialsSecretRef != nil {
		in, out := &in.CredentialsSecretRef, &out.CredentialsSecretRef
		*out = new(corev1.LocalObjectReference)
		**out = **in
	} else {
		out.CredentialsSecretRef = nil
	}
}

// DeepCopyInto copies AkeylessAuth into out.
func (in *AkeylessAuth) DeepCopyInto(out *AkeylessAuth) {
	if in.AccessKey != nil {
		out.AccessKey = new(AkeylessAuthAccessKey)
		*out.AccessKey = *in.AccessKey
	}
	if in.Kubernetes != nil {
		out.Kubernetes = new(AkeylessAuthKubernetes)
		*out.Kubernetes = *in.Kubernetes
	}
	if in.AwsIam != nil {
		out.AwsIam = new(AkeylessAuthAwsIam)
		in.AwsIam.DeepCopyInto(out.AwsIam)
	}
	if in.AzureAD != nil {
		out.AzureAD = new(AkeylessAuthAzureAD)
		in.AzureAD.DeepCopyInto(out.AzureAD)
	}
	if in.Gcp != nil {
		out.Gcp = new(AkeylessAuthGcp)
		in.Gcp.DeepCopyInto(out.Gcp)
	}
	if in.UniversalIdentity != nil {
		out.UniversalIdentity = new(AkeylessAuthUniversalIdentity)
		*out.UniversalIdentity = *in.UniversalIdentity
	}
	if in.Jwt != nil {
		out.Jwt = new(AkeylessAuthJwt)
		*out.Jwt = *in.Jwt
	}
}

// DeepCopyInto copies AkeylessAuthAwsIam into out.
func (in *AkeylessAuthAwsIam) DeepCopyInto(out *AkeylessAuthAwsIam) {
	*out = *in
	if in.CloudIDSecretRef != nil {
		in, out := &in.CloudIDSecretRef, &out.CloudIDSecretRef
		*out = new(corev1.LocalObjectReference)
		**out = **in
	} else {
		out.CloudIDSecretRef = nil
	}
}

// DeepCopyInto copies AkeylessAuthAzureAD into out.
func (in *AkeylessAuthAzureAD) DeepCopyInto(out *AkeylessAuthAzureAD) {
	*out = *in
	if in.CloudIDSecretRef != nil {
		in, out := &in.CloudIDSecretRef, &out.CloudIDSecretRef
		*out = new(corev1.LocalObjectReference)
		**out = **in
	} else {
		out.CloudIDSecretRef = nil
	}
}

// DeepCopyInto copies AkeylessAuthGcp into out.
func (in *AkeylessAuthGcp) DeepCopyInto(out *AkeylessAuthGcp) {
	*out = *in
	if in.CloudIDSecretRef != nil {
		in, out := &in.CloudIDSecretRef, &out.CloudIDSecretRef
		*out = new(corev1.LocalObjectReference)
		**out = **in
	} else {
		out.CloudIDSecretRef = nil
	}
}

// DeepCopy returns a deep copy of AkeylessIssuerSpec.
func (in *AkeylessIssuerSpec) DeepCopy() *AkeylessIssuerSpec {
	if in == nil {
		return nil
	}
	out := new(AkeylessIssuerSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto copies receiver into out.
func (in *AkeylessIssuer) DeepCopyInto(out *AkeylessIssuer) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status.Conditions = nil
	if in.Status.Conditions != nil {
		out.Status.Conditions = make([]v1.Condition, len(in.Status.Conditions))
		copy(out.Status.Conditions, in.Status.Conditions)
	}
}

// DeepCopy returns a deep copy of AkeylessIssuer.
func (in *AkeylessIssuer) DeepCopy() *AkeylessIssuer {
	if in == nil {
		return nil
	}
	out := new(AkeylessIssuer)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject returns a deep copy as runtime.Object.
func (in *AkeylessIssuer) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto copies receiver into out.
func (in *AkeylessIssuerList) DeepCopyInto(out *AkeylessIssuerList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]AkeylessIssuer, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy returns a deep copy of AkeylessIssuerList.
func (in *AkeylessIssuerList) DeepCopy() *AkeylessIssuerList {
	if in == nil {
		return nil
	}
	out := new(AkeylessIssuerList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject returns a deep copy as runtime.Object.
func (in *AkeylessIssuerList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto copies receiver into out.
func (in *AkeylessClusterIssuer) DeepCopyInto(out *AkeylessClusterIssuer) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status.Conditions = nil
	if in.Status.Conditions != nil {
		out.Status.Conditions = make([]v1.Condition, len(in.Status.Conditions))
		copy(out.Status.Conditions, in.Status.Conditions)
	}
}

// DeepCopy returns a deep copy of AkeylessClusterIssuer.
func (in *AkeylessClusterIssuer) DeepCopy() *AkeylessClusterIssuer {
	if in == nil {
		return nil
	}
	out := new(AkeylessClusterIssuer)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject returns a deep copy as runtime.Object.
func (in *AkeylessClusterIssuer) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto copies receiver into out.
func (in *AkeylessClusterIssuerList) DeepCopyInto(out *AkeylessClusterIssuerList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]AkeylessClusterIssuer, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy returns a deep copy of AkeylessClusterIssuerList.
func (in *AkeylessClusterIssuerList) DeepCopy() *AkeylessClusterIssuerList {
	if in == nil {
		return nil
	}
	out := new(AkeylessClusterIssuerList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject returns a deep copy as runtime.Object.
func (in *AkeylessClusterIssuerList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}
