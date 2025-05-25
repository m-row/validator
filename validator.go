package validator

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/m-row/finder"
	"github.com/m-row/validator/interfaces"

	"github.com/goccy/go-json"
	"github.com/google/uuid"
	js "github.com/santhosh-tekuri/jsonschema/v5"
)

var (
	ErrNotSupportedLocation = errors.New("not supported location")

	// KebabCase use as:
	//
	//	v.Check(
	//		validator.KebabCase.MatchString(val),
	//		key,
	//	    "invalid slug, must be of small letters and dashes only",
	//	)
	KebabCase = regexp.MustCompile("^[a-z0-9]+(?:-[a-z0-9]+)*$")
)

type KeyValue struct {
	Key   string `json:"key"`
	Value any    `json:"value"`
}

type Errors map[string][]string

// Validator type Define a new Validator type which contains a map of
// validation errors.
type Validator struct {
	T        interfaces.Translation
	Conn     finder.Connection
	QB       *squirrel.StatementBuilderType
	Schema   *js.Schema
	Scopes   []string
	Data     *Data
	Error    *js.ValidationError
	RootDIR  string
	DOMAIN   string
	newFile  string
	newImg   string
	newThumb string
	oldFile  *string
	oldImg   *string
	oldThumb *string
}

// NewValidator is a helper which creates a new Validator instance with an
// empty errors map.
func NewValidator(c *Config) (*Validator, error) {
	v := &Validator{
		T:       c.T,
		Conn:    c.Conn,
		QB:      c.QB,
		Schema:  c.Schema,
		Scopes:  c.Scopes,
		DOMAIN:  c.DOMAIN,
		RootDIR: c.RootDIR,
		Error: &js.ValidationError{
			KeywordLocation:         "",
			AbsoluteKeywordLocation: "",
			InstanceLocation:        "",
			Message:                 "",
			Causes:                  []*js.ValidationError{},
		},
	}
	if err := v.Parse(c.Request); err != nil {
		return nil, err
	}
	return v, nil
}

// ValidateModelSchema marshals model to json and validates it against schema
func (v *Validator) ValidateModelSchema(
	model any,
	tableName string,
	schema *js.Schema,
) {
	if schema == nil {
		v.AddModelSchemaError(
			tableName,
			errors.New("null validator schema for "+tableName),
		)
		return
	}
	data, err := json.Marshal(model)
	if err != nil {
		v.AddModelSchemaError(tableName, errors.New("couldn't marshal input"))
		return
	}
	if err := schema.Validate(data); err != nil {
		switch err := err.(type) { //nolint:errorlint // not of comparable err
		case *js.ValidationError:
			v.AddCause(err)
			return
		default:
			v.AddModelSchemaError(tableName, err)
			return
		}
	}
}

func (v *Validator) ValidatePropertySchema(key string) {
	if !v.Data.KeyExists(key) {
		v.Check(false, key, v.T.ValidateRequired())
		return
	}
	v.validateSchema(v.Data.GetBytes(key), key)
}

func (v *Validator) ValidateInterfaceSchema(i any) {
	data, err := json.Marshal(i)
	if err != nil {
		v.Check(false, "input", "couldn't unmarshal input")
		return
	}
	v.validateSchema(data, "input")
}

func (v *Validator) validateSchema(data []byte, key string) {
	var body any
	if err := json.Unmarshal(data, &body); err != nil {
		v.Check(false, key, "couldn't unmarshal input")
		return
	}
	if err := v.Schema.Validate(body); err != nil {
		switch err := err.(type) { //nolint:errorlint // not of comparable err
		case *js.ValidationError:
			v.AddCause(err)
		default:
			v.Check(false, key, err.Error())
		}
	}
}

func (v *Validator) AddModelSchemaError(tableName string, err error) {
	cause := &js.ValidationError{
		KeywordLocation:         "model/bind",
		AbsoluteKeywordLocation: "schema/validation",
		InstanceLocation:        tableName + "/model/bind",
		Message:                 err.Error(),
		Causes:                  []*js.ValidationError{},
	}
	v.Error.Causes = append(v.Error.Causes, cause)
}

func (v *Validator) Valid() bool {
	return v.Error.Message == "" && len(v.Error.Causes) == 0
}

func (v *Validator) AddCause(cause *js.ValidationError) {
	v.Error.Causes = append(v.Error.Causes, cause)
}

// Check the boolean condition, if !ok an error will be added to the causes
func (v *Validator) Check(ok bool, instanceLocation, message string) {
	if !ok {
		cause := &js.ValidationError{
			InstanceLocation: instanceLocation,
			Message:          message,
			Causes:           []*js.ValidationError{},
		}
		v.Error.Causes = append(v.Error.Causes, cause)
	}
}

func (v *Validator) GetErrorMap() Errors {
	errMap := make(Errors)
	errMap = v.loopCauses(errMap, v.Error.Causes)
	return errMap
}

func (v *Validator) loopCauses(
	errMap Errors,
	causes []*js.ValidationError,
) Errors {
	if len(causes) > 0 {
		for _, cause := range causes {
			key := cause.InstanceLocation
			key = strings.Replace(key, "/", "", 1)
			key = strings.ReplaceAll(key, "/", ".")
			message := cause.Message
			if len(cause.Causes) != 0 {
				errMap = v.loopCauses(errMap, cause.Causes)
			} else {
				if _, exists := errMap[key]; !exists {
					errMap[key] = []string{message}
				} else {
					errMap[key] = append(errMap[key], message)
				}
			}
		}
	}
	return errMap
}

func (v *Validator) AssignBool(
	key string,
	property *bool,
	allowedScopes ...string,
) {
	if v.Data.KeyExists(key) {
		v.Permit(key, allowedScopes)
		if property == nil {
			property = new(bool)
		}
		*property = v.Data.GetBool(key)
	}
}

func (v *Validator) AssignInt(
	key string,
	property *int,
	allowedScopes ...string,
) {
	if v.Data.KeyExists(key) {
		v.Permit(key, allowedScopes)
		if property == nil {
			property = new(int)
		}
		if value, err := strconv.ParseInt(v.Data.Get(key), 10, 0); err != nil {
			v.Check(false, key, v.T.ValidateInt())
		} else {
			*property = int(value)
		}
	}
}

func (v *Validator) AssignFloat(
	key string,
	property *float64,
	allowedScopes ...string,
) {
	if v.Data.KeyExists(key) {
		v.Permit(key, allowedScopes)
		if property == nil {
			property = new(float64)
		}
		if value, err := strconv.ParseFloat(v.Data.Get(key), 64); err != nil {
			v.Check(false, key, v.T.ValidateRequiredFloat())
		} else {
			*property = value
		}
	}
}

func (v *Validator) AssignDate(key string, property *string) *string {
	if v.Data.KeyExists(key) {
		if val := v.Data.Get(key); val != "" {
			if t, err := time.Parse(time.DateOnly, val); err != nil {
				v.Check(false, key, err.Error())
			} else {
				s := t.Format("2006-01-02")
				if property == nil {
					property = new(string)
				}
				*property = s
			}
		}
	}
	return property
}

func (v *Validator) AssignTimestamp(
	key string,
	property *time.Time,
	allowedScopes ...string,
) {
	if v.Data.KeyExists(key) {
		if val := v.Data.Get(key); val != "" {
			v.Permit(key, allowedScopes)
			t, err := time.Parse(time.RFC3339, val)
			if err != nil {
				v.Check(false, key, err.Error())
				return
			}
			if property != nil {
				*property = t
			}
		}
	}
}

func (v *Validator) AssignClock(
	key string,
	property *time.Time,
	allowedScopes ...string,
) {
	if v.Data.KeyExists(key) {
		if val := v.Data.Get(key); val != "" {
			v.Permit(key, allowedScopes)
			t, err := time.Parse("15:04", val)
			if err != nil {
				v.Check(false, key, err.Error())
				return
			}
			if property != nil {
				*property = t
			}
		}
	}
}

func (v *Validator) AssignUUID(
	key, fieldName, tableName string,
	property *uuid.UUID,
	required bool,
	allowedScopes ...string,
) *uuid.UUID {
	keyUUID := v.Data.GetUUID(key)
	if keyUUID != nil {
		v.Permit(key, allowedScopes)
		if property == nil {
			prop := uuid.Nil
			property = &prop
		}
		*property = *keyUUID
		v.Exists(property, key, fieldName, tableName, required)
	}
	return property
}

func (v *Validator) UnmarshalInto(
	key string,
	property any,
	allowedScopes ...string,
) {
	if v.Data.KeyExists(key) {
		v.Permit(key, allowedScopes)
		if err := v.Data.GetAndUnmarshalJSON(key, property); err != nil {
			v.Check(false, key, err.Error())
		}
	}
}

// Exists check if an id exists in a table row.
//
// using the following query:
//
//	SELECT EXISTS(SELECT 1 FROM tableName WHERE id=$1)
func (v *Validator) Exists(
	id any,
	key, tableField, tableName string,
	required bool,
) {
	var exists bool
	query := fmt.Sprintf(
		`SELECT EXISTS(SELECT 1 FROM %s WHERE %s=$1)`,
		tableName,
		tableField,
	)
	if err := v.Conn.GetContext(
		context.Background(),
		&exists,
		query,
		id,
	); err != nil {
		exists = false
	}
	if required {
		v.Check(exists, key, v.T.ValidateExistsInDB())
	}
}

// IDExistsInDB checks if the field value of an int id exists in database
func (v *Validator) IDExistsInDB(
	id *int,
	key, tableField, tableName string,
	required bool,
) {
	if id == nil && required {
		v.Check(false, key, v.T.ValidateRequired())
		return
	}
	// only allows the check if the value in the model is not equal to the input
	v.Exists(id, key, tableField, tableName, required)
}

// UUIDExistsInDB checks if the field value of a uuid exists in database.
func (v *Validator) UUIDExistsInDB(
	id *uuid.UUID,
	key, tableField, tableName string,
	required bool,
) {
	if id != nil {
		if required {
			v.Check(false, key, v.T.ValidateRequired())
		}
	}
	v.Exists(id, key, tableField, tableName, required)
}

// UserIDHasRole checks if the user id has role name associated with it
func (v *Validator) UserIDHasRole(
	fieldName string,
	userID *uuid.UUID,
	roleName string,
) {
	var exists bool

	query := `
		SELECT EXISTS(
		    SELECT 1
		    FROM users
                LEFT JOIN user_roles on users.id = user_roles."user_id"
                LEFT JOIN roles on roles.id = user_roles."role_id"
		    WHERE roles.name = $2
                AND users.id = $1
		) AS exists;
	`

	if err := v.Conn.GetContext(
		context.Background(),
		&exists,
		query,
		userID,
		roleName,
	); err != nil {
		exists = false
	}
	v.Check(exists, fieldName, v.T.ValidateMustHaveRole(roleName))
}
