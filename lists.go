package validator

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// ValidateListUUIDs unmarshalls a key to a string slice
func (v *Validator) ValidateListUUIDs(
	fieldName, tableName string,
	required bool,
	allowedScopes ...string,
) *[]string {
	arr := []string{}
	v.UnmarshalInto(fieldName, &arr, allowedScopes...)
	if required && len(arr) == 0 {
		v.Check(false, fieldName, v.T.ValidateRequiredArray())
	}
	if len(arr) > 0 {
		for index, id := range arr {
			if _, err := uuid.Parse(id); err != nil {
				v.Check(
					false,
					fmt.Sprintf("%s.%d", fieldName, index),
					v.T.ValidateUUID(),
				)
			} else {
				var exists bool
				query := fmt.Sprintf(
					`SELECT EXISTS(SELECT 1 FROM %s WHERE id=$1) AS exists`,
					tableName,
				)
				if err := v.Conn.GetContext(
					context.Background(),
					&exists,
					query,
					id,
				); err != nil {
					exists = false
				}
				if required && !exists {
					v.Check(
						exists,
						fmt.Sprintf("%s.%d", fieldName, index),
						v.T.ValidateExistsInDB(),
					)
				}
			}
		}
	}
	return &arr
}
