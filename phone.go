package validator

import (
	"context"
	"fmt"

	"github.com/ttacon/libphonenumber"
)

type PhoneCountry struct {
	ID        int    `json:"id"         db:"id"`
	ISO       string `json:"iso"        db:"iso"`
	PhoneCode int    `json:"phone_code" db:"phone_code"`
}

// ValidatePhone returns validated phone number in LY
// as region, code, national number
func (v *Validator) ValidatePhone(phone string) (*PhoneCountry, string) {
	var c PhoneCountry
	if phone == "" {
		v.Check(false, "phone", v.T.ValidateRequired())
		return nil, ""
	}

	if v.Data.KeyExists("region") {
		c.ISO = v.Data.Values.Get("region")
		found := false
		for k := range libphonenumber.GetSupportedRegions() {
			found = found || k == c.ISO
		}
		if !found {
			v.Check(false, "region", "no matching region for: "+c.ISO)
			return nil, ""
		}
	}
	if v.Data.Values.Has("country_code") {
		if err := v.Conn.GetContext(
			context.Background(),
			&c,
			` 
                SELECT id, iso, phone_code 
                FROM countries 
                WHERE phone_code = $1 OR iso = $2
            `,
			v.Data.Get("country_code"),
			c.ISO,
		); err != nil {
			v.Check(false, "country_code", err.Error())
			return nil, ""
		}
	}

	num, err := libphonenumber.Parse(phone, c.ISO)
	if err != nil {
		v.Check(false, "phone", err.Error())
		return nil, ""
	}

	cc := num.GetCountryCode()    // LY 218      | EG 20
	nn := num.GetNationalNumber() // 921234567   | 1001234567

	return &c, fmt.Sprintf("%d%d", cc, nn)
}
