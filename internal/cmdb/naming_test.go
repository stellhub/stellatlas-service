package cmdb

import "testing"

func TestValidateStandardAppID(t *testing.T) {
	naming, err := ValidateStandardAppID("stellaxis.payment.risk.antifraud.api")
	if err != nil {
		t.Fatalf("validate standard app id: %v", err)
	}
	if naming.Organization != "stellaxis" {
		t.Fatalf("expected organization stellaxis, got %q", naming.Organization)
	}
	if naming.BusinessDomain != "payment" {
		t.Fatalf("expected business domain payment, got %q", naming.BusinessDomain)
	}
	if naming.CapabilityDomain != "risk" {
		t.Fatalf("expected capability domain risk, got %q", naming.CapabilityDomain)
	}
	if naming.Application != "antifraud" {
		t.Fatalf("expected application antifraud, got %q", naming.Application)
	}
	if naming.Role != "api" {
		t.Fatalf("expected role api, got %q", naming.Role)
	}
}

func TestValidateStandardAppIDRejectsNonFiveSegmentName(t *testing.T) {
	if _, err := ValidateStandardAppID("payment.risk.antifraud.api"); err == nil {
		t.Fatal("expected non five segment name to be rejected")
	}
}

func TestValidateStandardAppIDRejectsUppercaseSegment(t *testing.T) {
	if _, err := ValidateStandardAppID("stellaxis.payment.risk.Antifraud.api"); err == nil {
		t.Fatal("expected uppercase segment to be rejected")
	}
}
