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

const (
	defaultRequestTimeout  = 2 * time.Minute
	defaultIssuanceTimeout = 2 * time.Minute
	asyncIssuanceTimeout   = 15 * time.Minute
	defaultPollInterval    = 15 * time.Second
)

// IssueOptions controls sync vs async (public CA) issuance behavior.
type IssueOptions struct {
	RequestTimeout  time.Duration
	IssuanceTimeout time.Duration
	AsyncIssuance   bool
	PollInterval    time.Duration
}

func (o IssueOptions) withDefaults() IssueOptions {
	if o.RequestTimeout <= 0 {
		o.RequestTimeout = defaultRequestTimeout
	}
	if o.PollInterval <= 0 {
		o.PollInterval = defaultPollInterval
	}
	if o.IssuanceTimeout <= 0 {
		if o.AsyncIssuance {
			o.IssuanceTimeout = asyncIssuanceTimeout
		} else {
			o.IssuanceTimeout = defaultIssuanceTimeout
		}
	}
	return o
}

// PendingError indicates public/async CA issuance is still in progress and should be retried slowly.
type PendingError struct {
	Err error
}

func (e PendingError) Error() string {
	if e.Err == nil {
		return "pki certificate issuance still pending"
	}
	return e.Err.Error()
}

func (e PendingError) Unwrap() error { return e.Err }

// IsPending reports whether err (or any wrapped error) is a PendingError.
func IsPending(err error) bool {
	var p PendingError
	return errors.As(err, &p)
}

// PermanentIssueError indicates a non-retryable signing failure (bad CSR, authz denied, etc.).
type PermanentIssueError struct {
	Err error
}

func (e PermanentIssueError) Error() string {
	if e.Err == nil {
		return "permanent pki certificate issuance error"
	}
	return e.Err.Error()
}

func (e PermanentIssueError) Unwrap() error { return e.Err }

// IsPermanent reports whether err (or any wrapped error) is a PermanentIssueError.
func IsPermanent(err error) bool {
	var p PermanentIssueError
	return errors.As(err, &p)
}

// IssueFromCSR calls Get PKI Certificate with a PEM CSR (cert-manager request bytes).
// ttlSeconds is ignored: sending cert-manager's requested duration often exceeds the PKI issuer
// default in Akeyless and causes HTTP 400; the signed cert lifetime follows the issuer config.
//
// When opts.AsyncIssuance is true, empty/pending responses are polled until IssuanceTimeout,
// and already-issued certificates are re-fetched via GetCertificateValue / destination-path lookup.
func IssueFromCSR(ctx context.Context, apiURL, token, certIssuerName string, csrPEM []byte, ttlSeconds int64, extendedKeyUsage string, opts IssueOptions) (leafPEM, parentPEM string, err error) {
	_ = ttlSeconds // see doc: not sent to API (issuer default lifetime)
	opts = opts.withDefaults()

	deadline := time.Now().Add(opts.IssuanceTimeout)
	ctx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()

	csrB64, err := csrDataBase64ForAkeyless(csrPEM)
	if err != nil {
		return "", "", PermanentIssueError{Err: err}
	}

	commonName, dnsNames := csrIdentity(csrPEM)
	client := newAPIClient(apiURL)

	// First issue attempt (may start public-CA ACME asynchronously on the gateway).
	leaf, parent, handle, callErr := getPKICertificateOnce(ctx, client, token, certIssuerName, csrB64, extendedKeyUsage, opts.RequestTimeout)
	if callErr == nil && leaf != "" {
		return leaf, parent, nil
	}

	lastErr := callErr
	if lastErr == nil {
		lastErr = fmt.Errorf("get-pki-certificate: empty certificate data in response")
	}
	if isPermanentPKIError(lastErr) && !isPendingPKIError(lastErr) {
		return "", "", PermanentIssueError{Err: lastErr}
	}
	if !opts.AsyncIssuance {
		return "", "", lastErr
	}

	issueHandle := handle
	attempt := 1
	for {
		if leaf2, _, fetchErr := tryRefetchCertificate(ctx, client, token, certIssuerName, issueHandle, commonName, dnsNames, opts.RequestTimeout); fetchErr == nil && leaf2 != "" {
			return leaf2, "", nil
		}

		if time.Now().After(deadline) || ctx.Err() != nil {
			return "", "", PendingError{Err: fmt.Errorf("async public CA issuance still pending after %d attempt(s): %w", attempt, lastErr)}
		}

		timer := time.NewTimer(opts.PollInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return "", "", PendingError{Err: fmt.Errorf("async public CA issuance canceled: %w", lastErr)}
		case <-timer.C:
		}

		attempt++
		// Re-issue sparingly: only when we have no handle to poll, or as a late last-resort.
		// Frequent re-issue against public CA can race ACME orders (orderNotReady).
		remaining := time.Until(deadline)
		shouldReissue := (issueHandle.path == "" && issueHandle.displayID == "" && issueHandle.readingToken == "") ||
			remaining <= 2*opts.PollInterval
		if !shouldReissue {
			continue
		}

		leaf, parent, handle, callErr = getPKICertificateOnce(ctx, client, token, certIssuerName, csrB64, extendedKeyUsage, opts.RequestTimeout)
		if callErr == nil && leaf != "" {
			return leaf, parent, nil
		}
		if handle.path != "" || handle.displayID != "" || handle.readingToken != "" {
			issueHandle = handle
		}
		if callErr != nil {
			lastErr = callErr
			if isPermanentPKIError(callErr) && !isPendingPKIError(callErr) {
				return "", "", PermanentIssueError{Err: callErr}
			}
		} else {
			lastErr = fmt.Errorf("get-pki-certificate: empty certificate data in response")
		}
	}
}

type certHandle struct {
	path         string
	displayID    string
	readingToken string
}

func newAPIClient(apiURL string) *akeyless.APIClient {
	return akeyless.NewAPIClient(&akeyless.Configuration{
		Servers: akeyless.ServerConfigurations{{URL: strings.TrimSuffix(apiURL, "/")}},
	})
}

func getPKICertificateOnce(ctx context.Context, client *akeyless.APIClient, token, certIssuerName, csrB64, extendedKeyUsage string, requestTimeout time.Duration) (leaf, parent string, handle certHandle, err error) {
	callCtx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	req := akeyless.NewGetPKICertificate(certIssuerName)
	req.SetToken(token)
	req.SetCsrDataBase64(csrB64)
	t := true
	req.SetJson(t)
	if extendedKeyUsage != "" {
		req.SetExtendedKeyUsage(extendedKeyUsage)
	}

	out, _, err := client.V2Api.GetPKICertificate(callCtx).Body(*req).Execute()
	if err != nil {
		return "", "", handle, enrichGetPKIError(err)
	}
	handle = certHandle{
		path:         strings.TrimSpace(out.GetPath()),
		displayID:    strings.TrimSpace(out.GetCertDisplayId()),
		readingToken: strings.TrimSpace(out.GetReadingToken()),
	}
	leaf = strings.TrimSpace(out.GetData())
	parent = strings.TrimSpace(out.GetParentCert())
	if leaf == "" {
		return "", parent, handle, nil
	}
	return leaf, parent, handle, nil
}

func tryRefetchCertificate(ctx context.Context, client *akeyless.APIClient, token, certIssuerName string, handle certHandle, commonName string, dnsNames []string, requestTimeout time.Duration) (leaf, parent string, err error) {
	if handle.path != "" || handle.displayID != "" {
		leaf, err = getCertificateValue(ctx, client, token, certIssuerName, handle, requestTimeout)
		if err == nil && leaf != "" {
			return leaf, "", nil
		}
	}

	destPath, derr := issuerDestinationPath(ctx, client, token, certIssuerName, requestTimeout)
	if derr != nil || destPath == "" {
		if err != nil {
			return "", "", err
		}
		return "", "", derr
	}

	name, lerr := findRecentCertificateName(ctx, client, token, destPath, commonName, dnsNames, requestTimeout)
	if lerr != nil || name == "" {
		if err != nil {
			return "", "", err
		}
		return "", "", lerr
	}
	handle.path = name
	leaf, err = getCertificateValue(ctx, client, token, certIssuerName, handle, requestTimeout)
	return leaf, "", err
}

func getCertificateValue(ctx context.Context, client *akeyless.APIClient, token, certIssuerName string, handle certHandle, requestTimeout time.Duration) (string, error) {
	callCtx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	name := handle.path
	if name == "" {
		name = "certificate"
	}
	req := akeyless.NewGetCertificateValue(name)
	req.SetToken(token)
	req.SetJson(true)
	req.SetIgnoreCache("true")
	if certIssuerName != "" {
		req.SetCertIssuerName(certIssuerName)
	}
	if handle.displayID != "" {
		req.SetDisplayId(handle.displayID)
	}
	if handle.readingToken != "" {
		req.SetIssuanceToken(handle.readingToken)
	}

	out, _, err := client.V2Api.GetCertificateValue(callCtx).Body(*req).Execute()
	if err != nil {
		return "", enrichGetPKIError(err)
	}
	pemData := strings.TrimSpace(out.GetCertificatePem())
	if pemData == "" {
		return "", fmt.Errorf("get-certificate-value: empty certificate_pem")
	}
	return pemData, nil
}

func issuerDestinationPath(ctx context.Context, client *akeyless.APIClient, token, certIssuerName string, requestTimeout time.Duration) (string, error) {
	callCtx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	req := akeyless.NewDescribeItem(certIssuerName)
	req.SetToken(token)
	req.SetJson(true)

	item, _, err := client.V2Api.DescribeItem(callCtx).Body(*req).Execute()
	if err != nil {
		return "", enrichGetPKIError(err)
	}
	details := item.GetCertificateIssueDetails()
	pki := details.GetPkiCertIssuerDetails()
	return strings.TrimSpace(pki.GetDestinationPath()), nil
}

func findRecentCertificateName(ctx context.Context, client *akeyless.APIClient, token, destPath, commonName string, dnsNames []string, requestTimeout time.Duration) (string, error) {
	callCtx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	filter := commonName
	if filter == "" && len(dnsNames) > 0 {
		filter = dnsNames[0]
	}

	req := akeyless.NewListItems()
	req.SetToken(token)
	req.SetJson(true)
	req.SetPath(destPath)
	req.SetMinimalView(false)
	if filter != "" {
		req.SetFilter(filter)
	}
	// Best-effort: certificates may appear under this type on some gateway versions.
	req.SetType([]string{"certificate"})

	out, _, err := client.V2Api.ListItems(callCtx).Body(*req).Execute()
	if err != nil {
		// Retry without type filter — some gateways reject "certificate" in type.
		req2 := akeyless.NewListItems()
		req2.SetToken(token)
		req2.SetJson(true)
		req2.SetPath(destPath)
		if filter != "" {
			req2.SetFilter(filter)
		}
		out, _, err = client.V2Api.ListItems(callCtx).Body(*req2).Execute()
		if err != nil {
			return "", enrichGetPKIError(err)
		}
	}

	items := out.GetItems()
	if len(items) == 0 {
		return "", fmt.Errorf("no certificates found under %q matching %q", destPath, filter)
	}

	want := make(map[string]struct{})
	if commonName != "" {
		want[strings.ToLower(commonName)] = struct{}{}
	}
	for _, d := range dnsNames {
		want[strings.ToLower(d)] = struct{}{}
	}

	var bestName string
	var bestTime time.Time
	for _, it := range items {
		name := strings.TrimSpace(it.GetItemName())
		if name == "" {
			continue
		}
		if len(want) > 0 {
			lname := strings.ToLower(name)
			matched := false
			for w := range want {
				if strings.Contains(lname, w) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}
		mod := it.GetModificationDate()
		if bestName == "" || mod.After(bestTime) {
			bestName = name
			bestTime = mod
		}
	}
	if bestName == "" {
		// Fall back to most recently modified item in the folder.
		for _, it := range items {
			name := strings.TrimSpace(it.GetItemName())
			mod := it.GetModificationDate()
			if name != "" && (bestName == "" || mod.After(bestTime)) {
				bestName = name
				bestTime = mod
			}
		}
	}
	if bestName == "" {
		return "", fmt.Errorf("no usable certificate item under %q", destPath)
	}
	return bestName, nil
}

func csrIdentity(csr []byte) (commonName string, dnsNames []string) {
	csr = bytes.TrimSpace(csr)
	var der []byte
	if blk, _ := pem.Decode(csr); blk != nil {
		der = blk.Bytes
	} else {
		der = csr
	}
	parsed, err := x509.ParseCertificateRequest(der)
	if err != nil {
		return "", nil
	}
	return parsed.Subject.CommonName, append([]string(nil), parsed.DNSNames...)
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

func isPendingPKIError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	pendingHints := []string{
		"invalid data in get pki certificate",
		"empty certificate data",
		"certificate not ready",
		"still pending",
		"issuance in progress",
		"orderNotReady",
		"order's status",
		"timeout",
		"deadline exceeded",
		"context deadline exceeded",
		"i/o timeout",
	}
	for _, h := range pendingHints {
		if strings.Contains(msg, strings.ToLower(h)) {
			return true
		}
	}
	return false
}

func isPermanentPKIError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	permanentHints := []string{
		"csr contains no data",
		"empty csr",
		"unauthorized",
		"access denied",
		"permission denied",
		"not allowed",
		"unsupported_top_level_domain",
		"certificate_unsupported",
		"invalid csr",
		"malformed",
		"status: 401",
		"status: 403",
		"http 401",
		"http 403",
	}
	for _, h := range permanentHints {
		if strings.Contains(msg, h) {
			return true
		}
	}
	return false
}
