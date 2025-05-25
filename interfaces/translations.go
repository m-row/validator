package interfaces

type Translation interface {
	ValidateRequired() string
	ValidateRequiredArray() string
	ValidateDate() string
	ValidateBool() string
	ValidateInt() string
	ValidateRequiredFloat() string
	ValidateUUID() string
	ValidateID() string
	ValidateExistsInDB() string
	ValidateNotExistsInDB() string
	ValidateMustBeInList(arg *[]string) string
	ValidateNotEmptyRoles() string
	ValidateMustHaveRole(role string) string
	ValidateMustBeGteZero() string
	ValidateMustBeGtZero() string
	ValidateMustBeLteValue(value int) string
	ValidateMinChar(value int) string
	ValidateMaxChar(value int) string
	ValidateMustBeGteFloatValue(value float64) string
	ValidateEmail() string
	ValidateStartWithLetter() string
	ValidateAlphanumericDashUnderscoreCharactersOnly() string
	ValidatePasswordConfirmationNoMatch() string
	ValidateCategoryInput() string
	ValidateCategoryParent() string
	UnDestroyableCategory() string
	UnsupportedLocation(name string) string
	NotPermitted(scopes, allowed []string) string
	UserAlreadyVerified() string
	FileIsNotAnImage() string

	ModelName(name string) string
	ModelNotFound(name string) string
	ModelDisabled(name string) string

	BadRequest() string
	ConflictError() string
	DeletedAccount() string
	DisabledAccount() string
	InputValidation() string
	InternalServerError() string
	InvalidCredentials() string
	JwtExpired() string
	LoggedOut() string
	MethodNotAllowed() string
	NotFound() string
	NotLoggedIn() string
	OutOfScopeError() string
	ProfileCleared() string
	UnauthorizedAccess() string

	OTPSentSuccessfully() string
	WalletTransactionAlreadyConfirmed() string
}
