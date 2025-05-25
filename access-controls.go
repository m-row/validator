package validator

import "slices"

// Permit checks the scopes array of request context
//
// it always allows admin to modify
//
// if now allowed array is provided it assumes anyone is allowed to modify
//
// otherwise it checks each allowed against context scopes
// and errors if no match is found
func (v *Validator) Permit(key string, allowed []string) {
	if allowed == nil {
		return
	}
	if len(allowed) == 0 {
		return
	}
	for i := range v.Scopes {
		if v.Scopes[i] == "admin" {
			return
		}
		if slices.Contains(allowed, v.Scopes[i]) {
			return
		}
	}
	v.Check(false, key, v.T.NotPermitted(v.Scopes, allowed))
}
