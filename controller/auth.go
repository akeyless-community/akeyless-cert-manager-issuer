package controller

import (
	"context"
	"fmt"
	"strings"

	akeyless "github.com/akeylesslabs/akeyless-go/v3"
	akeylessv1alpha1 "github.com/akeyless-community/akeyless-cert-manager-issuer/api/v1alpha1"
	cloudid "github.com/akeylesslabs/akeyless-go-cloud-id"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

func authBranchCount(a *akeylessv1alpha1.AkeylessAuth) int {
	if a == nil {
		return 0
	}
	n := 0
	if a.AccessKey != nil {
		n++
	}
	if a.Kubernetes != nil {
		n++
	}
	if a.AwsIam != nil {
		n++
	}
	if a.AzureAD != nil {
		n++
	}
	if a.Gcp != nil {
		n++
	}
	if a.UniversalIdentity != nil {
		n++
	}
	if a.Jwt != nil {
		n++
	}
	return n
}

func (s *Signer) buildAuthBody(ctx context.Context, spec akeylessv1alpha1.AkeylessIssuerSpec, secretNS string) (akeyless.Auth, error) {
	if spec.Auth == nil {
		if spec.CredentialsSecretRef == nil || spec.CredentialsSecretRef.Name == "" {
			return akeyless.Auth{}, fmt.Errorf("either spec.auth or spec.credentialsSecretRef must be set")
		}
		return s.authBodyAccessKey(ctx, secretNS, spec.CredentialsSecretRef.Name)
	}
	if n := authBranchCount(spec.Auth); n != 1 {
		return akeyless.Auth{}, fmt.Errorf("spec.auth must set exactly one authentication method, got %d", n)
	}
	switch {
	case spec.Auth.AccessKey != nil:
		if spec.Auth.AccessKey.SecretRef.Name == "" {
			return akeyless.Auth{}, fmt.Errorf("auth.accessKey.secretRef.name is required")
		}
		return s.authBodyAccessKey(ctx, secretNS, spec.Auth.AccessKey.SecretRef.Name)
	case spec.Auth.Kubernetes != nil:
		return s.authBodyKubernetes(ctx, secretNS, spec.Auth.Kubernetes)
	case spec.Auth.AwsIam != nil:
		return s.authBodyAwsIam(ctx, secretNS, spec.Auth.AwsIam)
	case spec.Auth.AzureAD != nil:
		return s.authBodyAzureAD(ctx, secretNS, spec.Auth.AzureAD)
	case spec.Auth.Gcp != nil:
		return s.authBodyGcp(ctx, secretNS, spec.Auth.Gcp)
	case spec.Auth.UniversalIdentity != nil:
		return s.authBodyUniversalIdentity(ctx, secretNS, spec.Auth.UniversalIdentity)
	case spec.Auth.Jwt != nil:
		return s.authBodyJwt(ctx, secretNS, spec.Auth.Jwt)
	default:
		return akeyless.Auth{}, fmt.Errorf("unsupported or empty spec.auth")
	}
}

func (s *Signer) authBodyAccessKey(ctx context.Context, ns, secretName string) (akeyless.Auth, error) {
	accessID, accessKey, err := s.readAccessKeySecret(ctx, ns, secretName)
	if err != nil {
		return akeyless.Auth{}, err
	}
	body := akeyless.NewAuthWithDefaults()
	body.SetAccessType("access_key")
	body.SetAccessId(accessID)
	body.SetAccessKey(accessKey)
	return *body, nil
}

func (s *Signer) authBodyKubernetes(ctx context.Context, ns string, k *akeylessv1alpha1.AkeylessAuthKubernetes) (akeyless.Auth, error) {
	if k.K8sAuthConfigName == "" {
		return akeyless.Auth{}, fmt.Errorf("auth.kubernetes.k8sAuthConfigName is required")
	}
	if k.ServiceAccountTokenSecretRef.Name == "" {
		return akeyless.Auth{}, fmt.Errorf("auth.kubernetes.serviceAccountTokenSecretRef.name is required")
	}
	tokKey := k.TokenSecretKey
	if tokKey == "" {
		tokKey = "token"
	}
	token, err := s.readSecretString(ctx, ns, k.ServiceAccountTokenSecretRef.Name, tokKey)
	if err != nil {
		return akeyless.Auth{}, err
	}
	body := akeyless.NewAuthWithDefaults()
	body.SetAccessType("k8s")
	if id := strings.TrimSpace(k.AccessID); id != "" {
		body.SetAccessId(id)
	}
	body.SetK8sAuthConfigName(k.K8sAuthConfigName)
	body.SetK8sServiceAccountToken(strings.TrimSpace(token))
	if g := strings.TrimSpace(k.GatewayURL); g != "" {
		body.SetGatewayUrl(strings.TrimSuffix(g, "/"))
	}
	return *body, nil
}

func (s *Signer) authBodyAwsIam(ctx context.Context, ns string, a *akeylessv1alpha1.AkeylessAuthAwsIam) (akeyless.Auth, error) {
	if strings.TrimSpace(a.AccessID) == "" {
		return akeyless.Auth{}, fmt.Errorf("auth.awsIam.accessId is required")
	}
	cid, err := s.resolveCloudID(ctx, ns, a.CloudID, a.CloudIDSecretRef, a.CloudIDSecretKey)
	if err != nil {
		return akeyless.Auth{}, err
	}
	body := akeyless.NewAuthWithDefaults()
	body.SetAccessType("aws_iam")
	body.SetAccessId(strings.TrimSpace(a.AccessID))
	body.SetCloudId(cid)
	return *body, nil
}

func (s *Signer) authBodyAzureAD(ctx context.Context, ns string, a *akeylessv1alpha1.AkeylessAuthAzureAD) (akeyless.Auth, error) {
	if strings.TrimSpace(a.AccessID) == "" {
		return akeyless.Auth{}, fmt.Errorf("auth.azureAd.accessId is required")
	}
	cid, err := s.resolveCloudID(ctx, ns, a.CloudID, a.CloudIDSecretRef, a.CloudIDSecretKey)
	if err != nil {
		return akeyless.Auth{}, err
	}
	body := akeyless.NewAuthWithDefaults()
	body.SetAccessType("azure_ad")
	body.SetAccessId(strings.TrimSpace(a.AccessID))
	body.SetCloudId(cid)
	return *body, nil
}

func (s *Signer) authBodyGcp(ctx context.Context, ns string, g *akeylessv1alpha1.AkeylessAuthGcp) (akeyless.Auth, error) {
	if strings.TrimSpace(g.AccessID) == "" {
		return akeyless.Auth{}, fmt.Errorf("auth.gcp.accessId is required")
	}
	cid, err := s.resolveCloudID(ctx, ns, g.CloudID, g.CloudIDSecretRef, g.CloudIDSecretKey)
	if err != nil {
		return akeyless.Auth{}, err
	}
	body := akeyless.NewAuthWithDefaults()
	body.SetAccessType("gcp")
	body.SetAccessId(strings.TrimSpace(g.AccessID))
	body.SetCloudId(cid)
	if aud := strings.TrimSpace(g.GcpAudience); aud != "" {
		body.SetGcpAudience(aud)
	}
	return *body, nil
}

func (s *Signer) authBodyUniversalIdentity(ctx context.Context, ns string, u *akeylessv1alpha1.AkeylessAuthUniversalIdentity) (akeyless.Auth, error) {
	if u.SecretRef.Name == "" {
		return akeyless.Auth{}, fmt.Errorf("auth.universalIdentity.secretRef.name is required")
	}
	key := u.UidTokenSecretKey
	if key == "" {
		key = "uid-token"
	}
	raw, err := s.readSecretString(ctx, ns, u.SecretRef.Name, key)
	if err != nil {
		return akeyless.Auth{}, err
	}
	body := akeyless.NewAuthWithDefaults()
	body.SetAccessType("universal_identity")
	body.SetUidToken(strings.TrimSpace(raw))
	return *body, nil
}

func (s *Signer) authBodyJwt(ctx context.Context, ns string, j *akeylessv1alpha1.AkeylessAuthJwt) (akeyless.Auth, error) {
	if j.SecretRef.Name == "" {
		return akeyless.Auth{}, fmt.Errorf("auth.jwt.secretRef.name is required")
	}
	key := j.JwtSecretKey
	if key == "" {
		key = "jwt"
	}
	raw, err := s.readSecretString(ctx, ns, j.SecretRef.Name, key)
	if err != nil {
		return akeyless.Auth{}, err
	}
	body := akeyless.NewAuthWithDefaults()
	body.SetAccessType("jwt")
	body.SetJwt(strings.TrimSpace(raw))
	if id := strings.TrimSpace(j.AccessID); id != "" {
		body.SetAccessId(id)
	}
	return *body, nil
}

func (s *Signer) resolveCloudID(ctx context.Context, ns, inline string, secRef *corev1.LocalObjectReference, key string) (string, error) {
	if s := strings.TrimSpace(inline); s != "" {
		return s, nil
	}
	if secRef != nil && secRef.Name != "" {
		if key == "" {
			key = "cloud-id"
		}
		v, err := s.readSecretString(ctx, ns, secRef.Name, key)
		if err != nil {
			return "", err
		}
		if strings.TrimSpace(v) == "" {
			return "", fmt.Errorf("secret %s/%s key %q is empty", ns, secRef.Name, key)
		}
		return strings.TrimSpace(v), nil
	}
	cid, err := cloudid.GetCloudId()
	if err != nil {
		return "", fmt.Errorf("cloud identity (set spec.cloudId or a secret, or run on AWS/GCP/Azure with metadata): %w", err)
	}
	return cid, nil
}

func (s *Signer) readSecretString(ctx context.Context, namespace, secretName, dataKey string) (string, error) {
	var sec corev1.Secret
	key := types.NamespacedName{Namespace: namespace, Name: secretName}
	if err := s.Client.Get(ctx, key, &sec); err != nil {
		if apierrors.IsNotFound(err) {
			return "", fmt.Errorf("secret %s/%s not found", namespace, secretName)
		}
		return "", fmt.Errorf("get secret %s/%s: %w", namespace, secretName, err)
	}
	b, ok := sec.Data[dataKey]
	if !ok || len(b) == 0 {
		return "", fmt.Errorf("secret %s/%s missing non-empty key %q", namespace, secretName, dataKey)
	}
	return string(b), nil
}

func (s *Signer) readAccessKeySecret(ctx context.Context, namespace, secretName string) (accessID, accessKey string, err error) {
	var sec corev1.Secret
	key := types.NamespacedName{Namespace: namespace, Name: secretName}
	if err := s.Client.Get(ctx, key, &sec); err != nil {
		if apierrors.IsNotFound(err) {
			return "", "", fmt.Errorf("secret %s/%s not found", namespace, secretName)
		}
		return "", "", fmt.Errorf("get secret %s/%s: %w", namespace, secretName, err)
	}
	idBytes, ok := sec.Data[secretKeyAccessID]
	if !ok || len(idBytes) == 0 {
		return "", "", fmt.Errorf("secret %s/%s missing key %q", namespace, secretName, secretKeyAccessID)
	}
	keyBytes, ok := sec.Data[secretKeyAccessKey]
	if !ok || len(keyBytes) == 0 {
		return "", "", fmt.Errorf("secret %s/%s missing key %q", namespace, secretName, secretKeyAccessKey)
	}
	return string(idBytes), string(keyBytes), nil
}
