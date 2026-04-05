package akeylessapi

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"
	"time"

	akeyless "github.com/akeylesslabs/akeyless-go/v3"
)

const authTimeout = 30 * time.Second

// IssueFromCSR calls Get PKI Certificate with a PEM CSR (cert-manager request bytes).
// ttlSeconds is ignored: sending cert-manager's requested duration often exceeds the PKI issuer
// default in Akeyless and causes HTTP 400; the signed cert lifetime follows the issuer config.
func IssueFromCSR(ctx context.Context, apiURL, token, certIssuerName string, csrPEM []byte, ttlSeconds int64, extendedKeyUsage string) (leafPEM, parentPEM string, err error) {
	_ = ttlSeconds // see doc: not sent to API (issuer default lifetime)
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	csrB64, err := csrDataBase64ForAkeyless(csrPEM)
	if err != nil {
		return "", "", err
	}

	client := akeyless.NewAPIClient(&akeyless.Configuration{
		Servers: akeyless.ServerConfigurations{{URL: strings.TrimSuffix(apiURL, "/")}},
	})

	req := akeyless.NewGetPKICertificate(certIssuerName)
	req.SetToken(token)
	req.SetCsrDataBase64(csrB64)
	t := true
	req.SetJson(t)
	if extendedKeyUsage != "" {
		req.SetExtendedKeyUsage(extendedKeyUsage)
	}

	out, _, err := client.V2Api.GetPKICertificate(ctx).Body(*req).Execute()
	if err != nil {
		return "", "", enrichGetPKIError(err)
	}
	leaf := strings.TrimSpace(out.GetData())
	if leaf == "" {
		return "", "", fmt.Errorf("get-pki-certificate: empty certificate data in response")
	}
	parent := strings.TrimSpace(out.GetParentCert())
	return leaf, parent, nil
}

// csrDataBase64ForAkeyless returns csr-data-base64 for /get-pki-certificate.
// Akeyless expects the CSR PEM file (or raw PKCS#10 DER) base64-encoded — not base64(DER) alone.
// Sending only DER caused HTTP 400 "csr contains no data" against the gateway.
func csrDataBase64ForAkeyless(csr []byte) (string, error) {
	csr = bytes.TrimSpace(csr)
	if len(csr) == 0 {
		return "", fmt.Errorf("empty CSR")
	}
	if blk, _ := pem.Decode(csr); blk != nil {
		switch blk.Type {
		case "CERTIFICATE REQUEST", "NEW CERTIFICATE REQUEST":
			var buf bytes.Buffer
			if err := pem.Encode(&buf, blk); err != nil {
				return "", fmt.Errorf("encode CSR PEM: %w", err)
			}
			return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
		}
	}
	if _, err := x509.ParseCertificateRequest(csr); err == nil {
		return base64.StdEncoding.EncodeToString(csr), nil
	}
	return "", fmt.Errorf("CSR must be PEM CERTIFICATE REQUEST (or PKCS#10 DER)")
}

func enrichGetPKIError(err error) error {
	var apiErr akeyless.GenericOpenAPIError
	if errors.As(err, &apiErr) {
		detail := strings.TrimSpace(string(apiErr.Body()))
		if m := apiErr.Model(); m != nil {
			if je, ok := m.(akeyless.JSONError); ok {
				if s := strings.TrimSpace(je.GetError()); s != "" {
					detail = s
				}
			}
		}
		if detail != "" {
			return fmt.Errorf("get-pki-certificate: %v — %s", err, detail)
		}
	}
	return fmt.Errorf("get-pki-certificate: %w", err)
}
