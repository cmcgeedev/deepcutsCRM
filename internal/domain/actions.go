package domain

import (
	"bytes"
	"regexp"
)

type ActionType string

const (
	ActionDeliver ActionType = "deliver"
	ActionAdjust  ActionType = "adjust"
	ActionSkip    ActionType = "skip"
)

type ProofType string

const (
	ProofSignature ProofType = "signature"
	ProofPhoto     ProofType = "photo"
	ProofName      ProofType = "name"
)

// MaxProofBytes caps a decoded signature or photo so the offline queue stays small.
const MaxProofBytes = 300 * 1024

type Proof struct {
	Type ProofType
	Data []byte // PNG or JPEG bytes for signature/photo
	Name string // received-by name for ProofName
}

type LineAdjustment struct {
	LineID          int64
	DeliveredQty    *Hundredths
	DeliveredWeight *Hundredths
	ShortageNote    *string
}

type DriverAction struct {
	ClientID   string
	Type       ActionType
	Proof      *Proof
	Lines      []LineAdjustment
	Note       string
	SkipReason string
}

var uuidRe = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func (a DriverAction) Validate() map[string]string {
	errs := map[string]string{}
	if !uuidRe.MatchString(a.ClientID) {
		errs["clientId"] = "must be a UUID"
	}
	switch a.Type {
	case ActionDeliver:
		if a.Proof == nil {
			errs["proof"] = "required to deliver"
		} else if msg := a.Proof.validate(); msg != "" {
			errs["proof"] = msg
		}
	case ActionAdjust:
		if len(a.Lines) == 0 {
			errs["lines"] = "at least one line adjustment required"
		}
	case ActionSkip:
		if a.SkipReason == "" {
			errs["skipReason"] = "required to skip"
		}
	default:
		errs["type"] = "must be deliver, adjust or skip"
	}
	for _, l := range a.Lines {
		if (l.DeliveredQty != nil && *l.DeliveredQty < 0) || (l.DeliveredWeight != nil && *l.DeliveredWeight < 0) {
			errs["lines"] = "delivered quantities must not be negative"
		}
	}
	return errs
}

func (p Proof) validate() string {
	switch p.Type {
	case ProofName:
		if p.Name == "" {
			return "received-by name required"
		}
	case ProofSignature, ProofPhoto:
		if len(p.Data) == 0 {
			return "image data required"
		}
		if len(p.Data) > MaxProofBytes {
			return "image larger than 300 KB"
		}
		if !isPNGOrJPEG(p.Data) {
			return "image must be PNG or JPEG"
		}
	default:
		return "type must be signature, photo or name"
	}
	return ""
}

// isPNGOrJPEG sniffs the magic bytes of a signature/photo proof image.
func isPNGOrJPEG(data []byte) bool {
	return bytes.HasPrefix(data, []byte{0x89, 'P', 'N', 'G'}) || bytes.HasPrefix(data, []byte{0xFF, 0xD8, 0xFF})
}
