package idpworkflow

import (
	"encoding/json"
	"time"

	"github.com/pkg/errors"
)

// DecodePresentation decodes the wire representation returned by
// ctx.present.form. Duration stays out of JavaScript: the wire format uses a
// bounded integer number of seconds and native validation enforces the policy.
func DecodePresentation(encoded json.RawMessage) (Presentation, error) {
	var wire struct {
		Title            string             `json:"title"`
		ResumeHandler    string             `json:"resumeHandler"`
		Fields           []FieldID          `json:"fields"`
		Actions          []ActionID         `json:"actions"`
		PublicValues     map[FieldID]string `json:"publicValues"`
		Errors           []FieldError       `json:"errors"`
		Carry            json.RawMessage    `json:"carry"`
		ExpiresInSeconds int64              `json:"expiresInSeconds"`
	}
	if len(encoded) == 0 {
		return Presentation{}, errors.New("workflow presentation is missing")
	}
	if err := json.Unmarshal(encoded, &wire); err != nil {
		return Presentation{}, errors.Wrap(err, "decode workflow presentation")
	}
	if wire.ExpiresInSeconds <= 0 || wire.ExpiresInSeconds > int64(DefaultMaximumContinuationTTL/time.Second) {
		return Presentation{}, errors.New("workflow presentation has invalid expiry")
	}
	return Presentation{
		Title: wire.Title, ResumeHandler: wire.ResumeHandler, Fields: wire.Fields,
		Actions: wire.Actions, PublicValues: wire.PublicValues, Errors: wire.Errors,
		Carry: wire.Carry, ExpiresIn: time.Duration(wire.ExpiresInSeconds) * time.Second,
	}, nil
}
