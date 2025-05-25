package validator

import (
	"strings"
	"unicode"
)

// isAlphanumericDashSpaceOrUnderscore self explanatory
func isAlphanumericDashSpaceOrUnderscore(r rune) bool {
	return unicode.IsLetter(r) ||
		unicode.Is(unicode.Number, r) ||
		unicode.IsSpace(r) ||
		r == '-' || r == '_'
}

func allowedAlphanumericDashAndUnderscores(s string) bool {
	success := true
	for _, r := range s {
		success = success && isAlphanumericDashSpaceOrUnderscore(r)
		if !success {
			break
		}
	}
	return success
}

// startsWithLetter checks if the string starts
// with an Arabic or English letter.
func startsWithLetter(s string) bool {
	for _, r := range s {
		return unicode.IsLetter(r)
	}
	return false
}

// AssignString to allow only admin to modify attribute:
//
//	v.AssignString("name", &m.Name, "admin")
//
// to allow anyone to modify:
//
//	v.AssignString("name", &m.Name)
//
// to pass specific scopes, admin will still be permitted:
//
//	v.AssignString("name", &m.Name, "vendor", "driver")
//
// nullable strings must be assigned back:
//
//	m.Name = v.AssignString("name", m.Name)
func (v *Validator) AssignString(
	key string,
	property *string,
	minlength, maxlength int,
	allowedScopes ...string,
) *string {
	if v.Data.KeyExists(key) {
		v.Permit(key, allowedScopes)
		if val := v.Data.Values.Get(key); val != "" {
			if property == nil {
				property = new(string)
			}
			// removes consecutive white spaces in between
			// + leading and trailing
			*property = strings.Join(
				strings.Fields(strings.TrimSpace(val)),
				" ",
			)
			if len(*property) < minlength {
				v.Check(false, key, v.T.ValidateMinChar(minlength))
				return nil
			}
			if len(*property) > maxlength {
				v.Check(false, key, v.T.ValidateMaxChar(maxlength))
				return nil
			}
		}
	}
	return property
}

// ParseString checks if the string starts with a letter and is composed of
// only alphanumerics, dash and underscores
func (v *Validator) ParseString(
	key string,
	property *string,
) {
	if property == nil {
		v.Check(false, key, v.T.ValidateRequired())
		return
	}

	*property = strings.Join(
		strings.Fields(strings.TrimSpace(*property)),
		" ",
	)
	if !startsWithLetter(*property) {
		v.Check(false, key, v.T.ValidateStartWithLetter())
		return
	}

	if !allowedAlphanumericDashAndUnderscores(*property) {
		v.Check(
			false,
			key,
			v.T.ValidateAlphanumericDashUnderscoreCharactersOnly(),
		)
		return
	}
}
