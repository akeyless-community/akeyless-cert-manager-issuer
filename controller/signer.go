package controller

import (
	"context"
	"crypto/x509"
	"fmt"
	"strings"
	"time"

	issuerv1alpha1 "github.com/cert-manager/issuer-lib/api/v1alpha1"
	"github.com/cert-manager/issuer-lib/controllers"
	"github.com/cert-manager/issuer-lib/controllers/signer"
	"github.com/cert-manager/cert-manager/pkg/util/pki"
	akeylessv1alpha1 "github.com/akeyless-community/akeyless-cert-manager-issuer/api/v1alpha1"
	"github.com/akeyless-community/akeyless-cert-manager-issuer/internal/akeylessapi"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	secretKeyAccessID  = "access-id"
	secretKeyAccessKey = "access-key"
)

// Signer reconciles AkeylessIssuer / AkeylessClusterIssuer and signs CertificateRequests via the Akeyless API.
type Signer struct {
	Client                   client.Client
	ClusterResourceNamespace string
}

// +kubebuilder:rbac:groups=cert-manager.io,resources=certificaterequests,verbs=get;list;watch
// +kubebuilder:rbac:groups=cert-manager.io,resources=certificaterequests/status,verbs=patch
// +kubebuilder:rbac:groups=certificates.k8s.io,resources=certificatesigningrequests,verbs=get;list;watch
// +kubebuilder:rbac:groups=certificates.k8s.io,resources=certificatesigningrequests/status,verbs=patch
// +kubebuilder:rbac:groups=certificates.k8s.io,resources=signers,verbs=sign,resourceNames=akeylessissuers.certmanager.akeyless.io/*;akeylessclusterissuers.certmanager.akeyless.io/*
// +kubebuilder:rbac:groups=certmanager.akeyless.io,resources=akeylessissuers;akeylessclusterissuers,verbs=get;list;watch
// +kubebuilder:rbac:groups=certmanager.akeyless.io,resources=akeylessissuers/status;akeylessclusterissuers/status,verbs=patch
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch

func (s *Signer) SetupWithManager(ctx context.Context, mgr ctrl.Manager, maxRetry time.Duration) error {
	s.Client = mgr.GetClient()
	return (&controllers.CombinedController{
		IssuerTypes:        []issuerv1alpha1.Issuer{&akeylessv1alpha1.AkeylessIssuer{}},
		ClusterIssuerTypes: []issuerv1alpha1.Issuer{&akeylessv1alpha1.AkeylessClusterIssuer{}},
		FieldOwner:         "akeyless-cert-manager-issuer.certmanager.akeyless.io",
		MaxRetryDuration:   maxRetry,
		Sign:               s.Sign,
		Check:              s.Check,
		EventRecorder:      mgr.GetEventRecorder("akeyless-cert-manager-issuer.certmanager.akeyless.io"),
	}).SetupWithManager(ctx, mgr)
}

func (s *Signer) Check(ctx context.Context, issuerObject issuerv1alpha1.Issuer) error {
	spec, secretNS, err := s.resolveIssuer(issuerObject)
	if err != nil {
		return signer.PermanentError{Err: err}
	}
	body, err := s.buildAuthBody(ctx, spec, secretNS)
	if err != nil {
		return signer.PermanentError{Err: err}
	}
	_, err = akeylessapi.Authenticate(ctx, spec.APIURL, body)
	if err != nil {
		return fmt.Errorf("akeyless authentication failed: %w", err)
	}
	return nil
}

func (s *Signer) Sign(ctx context.Context, cr signer.CertificateRequestObject, issuerObject issuerv1alpha1.Issuer) (signer.PEMBundle, error) {
	spec, secretNS, err := s.resolveIssuer(issuerObject)
	if err != nil {
		return signer.PEMBundle{}, signer.PermanentError{Err: err}
	}
	body, err := s.buildAuthBody(ctx, spec, secretNS)
	if err != nil {
		return signer.PEMBundle{}, signer.PermanentError{Err: err}
	}
	token, err := akeylessapi.Authenticate(ctx, spec.APIURL, body)
	if err != nil {
		return signer.PEMBundle{}, signer.IssuerError{Err: fmt.Errorf("authenticate: %w", err)}
	}

	details, err := cr.GetCertificateDetails()
	if err != nil {
		return signer.PEMBundle{}, signer.PermanentError{Err: err}
	}
	ttl := int64(details.Duration / time.Second)
	if ttl < 1 {
		ttl = 0
	}
	eku := extKeyUsageCSV(details.ExtKeyUsage)

	leaf, parent, err := akeylessapi.IssueFromCSR(ctx, spec.APIURL, token, spec.CertIssuerName, details.CSR, ttl, eku)
	if err != nil {
		return signer.PEMBundle{}, err
	}

	chain := ensurePEMNewline(leaf)
	if parent != "" {
		chain += ensurePEMNewline(parent)
	}

	var caPEM []byte
	if parent != "" {
		caPEM = []byte(ensurePEMNewline(parent))
	}

	return signer.PEMBundle(pki.PEMBundle{
		ChainPEM: []byte(chain),
		CAPEM:    caPEM,
	}), nil
}

func (s *Signer) resolveIssuer(issuerObject issuerv1alpha1.Issuer) (spec akeylessv1alpha1.AkeylessIssuerSpec, secretNamespace string, err error) {
	switch iss := issuerObject.(type) {
	case *akeylessv1alpha1.AkeylessIssuer:
		return iss.Spec, iss.Namespace, nil
	case *akeylessv1alpha1.AkeylessClusterIssuer:
		if s.ClusterResourceNamespace == "" {
			return spec, "", fmt.Errorf("cluster issuer requires controller --cluster-resource-namespace")
		}
		return iss.Spec, s.ClusterResourceNamespace, nil
	default:
		return spec, "", fmt.Errorf("unsupported issuer type %T", issuerObject)
	}
}

func extKeyUsageCSV(usages []x509.ExtKeyUsage) string {
	var parts []string
	for _, u := range usages {
		switch u {
		case x509.ExtKeyUsageServerAuth:
			parts = append(parts, "serverauth")
		case x509.ExtKeyUsageClientAuth:
			parts = append(parts, "clientauth")
		}
	}
	return strings.Join(parts, ",")
}

func ensurePEMNewline(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if !strings.HasSuffix(s, "\n") {
		return s + "\n"
	}
	return s
}
