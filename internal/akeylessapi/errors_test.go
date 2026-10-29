package akeylessapi

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestIsPendingPKIError(t *testing.T) {
	cases := []struct {
		msg  string
		want bool
	}{
		{"invalid data in get pki certificate response", true},
		{"get-pki-certificate: empty certificate data in response", true},
		{"context deadline exceeded", true},
		{"orderNotReady", true},
		{"csr contains no data", false},
		{"access denied", false},
		{"", false},
	}
	for _, tc := range cases {
		got := isPendingPKIError(errors.New(tc.msg))
		if got != tc.want {
			t.Fatalf("isPendingPKIError(%q)=%v want %v", tc.msg, got, tc.want)
		}
	}
}

func TestIsPermanentPKIError(t *testing.T) {
	cases := []struct {
		msg  string
		want bool
	}{
		{"csr contains no data", true},
		{"certificate_unsupported_top_level_domain", true},
		{"unauthorized", true},
		{"invalid data in get pki certificate response", false},
		{"empty certificate data", false},
	}
	for _, tc := range cases {
		got := isPermanentPKIError(errors.New(tc.msg))
		if got != tc.want {
			t.Fatalf("isPermanentPKIError(%q)=%v want %v", tc.msg, got, tc.want)
		}
	}
}

func TestPendingAndPermanentWrappers(t *testing.T) {
	p := PendingError{Err: fmt.Errorf("still pending")}
	if !IsPending(p) || IsPermanent(p) {
		t.Fatalf("PendingError classification failed")
	}
	perm := PermanentIssueError{Err: fmt.Errorf("bad csr")}
	if !IsPermanent(perm) || IsPending(perm) {
		t.Fatalf("PermanentIssueError classification failed")
	}
}

func TestIssueOptionsDefaults(t *testing.T) {
	syncOpts := IssueOptions{}.withDefaults()
	if syncOpts.RequestTimeout != 2*time.Minute || syncOpts.IssuanceTimeout != 2*time.Minute {
		t.Fatalf("sync defaults: %+v", syncOpts)
	}
	asyncOpts := IssueOptions{AsyncIssuance: true}.withDefaults()
	if asyncOpts.IssuanceTimeout != 15*time.Minute {
		t.Fatalf("async default issuanceTimeout=%s want 15m", asyncOpts.IssuanceTimeout)
	}
}
