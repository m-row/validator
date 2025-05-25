package validator

func AssignENUM[T ~string](
	v *Validator,
	key string,
	property *T,
	allowedScopes ...string,
) *T {
	if v.Data.KeyExists(key) {
		v.Permit(key, allowedScopes)
		if val := v.Data.Values.Get(key); val != "" {
			if property == nil {
				property = new(T)
			}
			*property = T(val)
		}
	}
	return property
}

func UnmarshalIntoNullable[T any](
	v *Validator,
	key string,
	property *T,
	allowedScopes ...string,
) *T {
	if v.Data.KeyExists(key) {
		v.Permit(key, allowedScopes)
		if property == nil {
			property = new(T)
		}
		if err := v.Data.GetAndUnmarshalJSON(key, property); err != nil {
			v.Check(false, key, err.Error())
		}
	}
	return property
}
