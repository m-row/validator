package interfaces

// HasImage interface handles img and thumb fields in a model.
type HasImage interface {
	// GetID returns id of model as a string
	GetID() string
	// TableName returns text name of table of model to
	// reference the directory in uploads
	TableName() string
	// GetImg returns the current assigned string value for img
	GetImg() *string
	// SetImg assigns value of img to the model
	SetImg(name *string)
	// GetThumb returns the current assigned string value for thumb
	GetThumb() *string
	// SetThumb assigns value of thumb to the model
	SetThumb(name *string)
}
