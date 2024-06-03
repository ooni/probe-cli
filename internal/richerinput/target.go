package richerinput

//
// Definition of target and related types
//

import "github.com/ooni/probe-cli/v3/internal/model"

// Target is an alias for [model.RicherInputTarget].
type Target = model.RicherInputTarget

// Void represents the absence of richer input.
type VoidTarget struct{}

var _ Target = VoidTarget{}

// CategoryCode implements model.RicherInputTarget.
func (v VoidTarget) CategoryCode() string {
	return model.DefaultCategoryCode
}

// CountryCode implements model.RicherInputTarget.
func (v VoidTarget) CountryCode() string {
	return model.DefaultCountryCode
}

// Input implements Target.
func (v VoidTarget) Input() model.MeasurementInput {
	return ""
}

// Options implements Target.
func (v VoidTarget) Options() []string {
	return nil
}
