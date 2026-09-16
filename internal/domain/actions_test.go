package domain

import (
	"strings"
	"testing"
)

const uuid1 = "6f1c2f1e-5b7a-4c1e-9c3a-1d2e3f4a5b6c"

func TestDriverActionValidate(t *testing.T) {
	ok := DriverAction{ClientID: uuid1, Type: ActionDeliver, Proof: &Proof{Type: ProofName, Name: "Pat"}}
	if errs := ok.Validate(); len(errs) != 0 {
		t.Fatalf("unexpected: %v", errs)
	}
	if errs := (DriverAction{ClientID: "nope", Type: ActionDeliver, Proof: &Proof{Type: ProofName, Name: "P"}}).Validate(); errs["clientId"] == "" {
		t.Error("bad uuid must fail")
	}
	if errs := (DriverAction{ClientID: uuid1, Type: "fly"}).Validate(); errs["type"] == "" {
		t.Error("bad type must fail")
	}
	if errs := (DriverAction{ClientID: uuid1, Type: ActionDeliver}).Validate(); errs["proof"] == "" {
		t.Error("deliver without proof must fail")
	}
	if errs := (DriverAction{ClientID: uuid1, Type: ActionDeliver, Proof: &Proof{Type: ProofSignature}}).Validate(); errs["proof"] == "" {
		t.Error("signature without data must fail")
	}
	big := &Proof{Type: ProofPhoto, Data: []byte(strings.Repeat("x", MaxProofBytes+1))}
	if errs := (DriverAction{ClientID: uuid1, Type: ActionDeliver, Proof: big}).Validate(); errs["proof"] == "" {
		t.Error("oversized proof must fail")
	}
	notImage := &Proof{Type: ProofSignature, Data: []byte("not an image")}
	if errs := (DriverAction{ClientID: uuid1, Type: ActionDeliver, Proof: notImage}).Validate(); errs["proof"] == "" {
		t.Error("non-PNG/JPEG proof data must fail")
	}
	if errs := (DriverAction{ClientID: uuid1, Type: ActionSkip}).Validate(); errs["skipReason"] == "" {
		t.Error("skip without reason must fail")
	}
	if errs := (DriverAction{ClientID: uuid1, Type: ActionAdjust}).Validate(); errs["lines"] == "" {
		t.Error("adjust without lines must fail")
	}
	neg := Hundredths(-1)
	if errs := (DriverAction{ClientID: uuid1, Type: ActionAdjust, Lines: []LineAdjustment{{LineID: 1, DeliveredQty: &neg}}}).Validate(); errs["lines"] == "" {
		t.Error("negative delivered qty must fail")
	}
}
