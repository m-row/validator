package validator

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/goccy/go-json"
	"github.com/google/uuid"
)

const DefaultMaxFormSize = 16 << 20 // 16mb

// Data holds data obtained from the request body and url query parameters.
// Because Data is built from multiple sources, sometimes there will be more
// than one value for a given key. You can use Get, Set, Add, and Del to access
// the first element for a given key or access the Values and Files properties
// directly to access additional elements for a given key. You can also use
// helper methods to convert the first value for a given key to a different
// type (e.g. bool or int).
type Data struct {
	// Values holds any basic key-value string data
	// This includes all fields from a json body or
	// urlencoded form, and the form fields only (not
	// files) from a multipart form
	Values url.Values
	// Files holds files from a multipart form only.
	// For any other type of request, it will always
	// be empty. Files only supports one file per key,
	// since this is by far the most common use. If you
	// need to have more than one file per key, parse the
	// files manually using req.MultipartForm.File.
	Files map[string]*multipart.FileHeader
	// jsonBody holds the original body of the request.
	// Only available for json requests.
	jsonBody []byte
}

func newData() *Data {
	return &Data{
		Values: url.Values{},
		Files:  map[string]*multipart.FileHeader{},
	}
}

// Parse uses the default max form size defined above and calls ParseMax.
func (v *Validator) Parse(req *http.Request) error {
	data, err := v.ParseMax(req, DefaultMaxFormSize)
	if err != nil {
		return err
	}
	v.Data = data
	return nil
}

func (v *Validator) ParseMax(
	req *http.Request,
	maxMemory int64,
) (*Data, error) {
	data := newData()
	contentType := req.Header.Get("Content-Type")
	switch contentType {
	case "multipart/form-data":
		if err := req.ParseMultipartForm(maxMemory); err != nil {
			return nil, err
		}
		for key, vals := range req.MultipartForm.Value {
			for _, val := range vals {
				data.Add(key, val)
			}
		}
		for key, files := range req.MultipartForm.File {
			if len(files) != 0 {
				data.AddFile(key, files[0])
			}
		}
	case "form-urlencoded":
		if err := req.ParseForm(); err != nil {
			return nil, err
		}
		for key, vals := range req.PostForm {
			for _, val := range vals {
				data.Add(key, val)
			}
		}
	case "application/json":
		body, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		data.jsonBody = body
		if err := parseJSON(data.Values, data.jsonBody); err != nil {
			return nil, err
		}
	}
	for key, vals := range req.URL.Query() {
		for _, val := range vals {
			data.Add(key, val)
		}
	}
	return data, nil
}

// CreateFromMap returns a Data object with keys and values matching
// the map.
func CreateFromMap(m map[string]string) *Data {
	data := newData()
	for key, value := range m {
		data.Add(key, value)
	}
	return data
}

func parseJSON(values url.Values, body []byte) error {
	if len(body) == 0 {
		// don't attempt to parse empty bodies
		return nil
	}
	rawData := map[string]any{}
	if err := json.Unmarshal(body, &rawData); err != nil {
		return err
	}
	// Whatever the underlying type is, we need to convert it to a
	// string. There are only a few possible types, so we can just
	// do a type switch over the possibilities.
	for key, val := range rawData {
		switch val.(type) {
		case string, bool, float64:
			values.Add(key, fmt.Sprint(val))
		case nil:
			values.Add(key, "")
		case map[string]any, []any:
			// for more complicated data structures, convert back to
			// a JSON string and let user decide how to unmarshal
			jsonVal, err := json.Marshal(val)
			if err != nil {
				return err
			}
			values.Add(key, string(jsonVal))
		}
	}
	return nil
}

// Add adds the value to key.
// It appends to any existing values associated with key.
func (d *Data) Add(key, value string) {
	d.Values.Add(key, value)
}

// AddFile adds the multipart form file to data with the given key.
func (d *Data) AddFile(key string, file *multipart.FileHeader) {
	d.Files[key] = file
}

// Del deletes the values associated with key.
func (d *Data) Del(key string) {
	d.Values.Del(key)
}

// DelFile deletes the file associated with key (if any).
// If there is no file associated with key, it does nothing.
func (d *Data) DelFile(key string) {
	delete(d.Files, key)
}

// Encode encodes the values into “URL encoded” form ("bar=baz&foo=quux")
// sorted by key. Any files in d will be ignored because there is no direct way
// to convert a file to a URL encoded value.
func (d *Data) Encode() string {
	return d.Values.Encode()
}

// Get gets the first value associated with the given key. If there are no
// values associated with the key, Get returns the empty string.
// To access multiple values, use the map directly.
func (d Data) Get(key string) string {
	return d.Values.Get(key)
}

// GetFile returns the multipart form file associated with key, if any, as a
// *multipart.FileHeader. If there is no file associated with key, it returns
// nil. If you just want the body of the file, use GetFileBytes.
func (d Data) GetFile(key string) *multipart.FileHeader {
	return d.Files[key]
}

// Set sets the key to value. It replaces any existing values.
func (d *Data) Set(key, value string) {
	d.Values.Set(key, value)
}

// KeyExists returns true if data.Values[key] exists. When parsing a request
// body, the key is considered to be in existence if it was provided in the
// request body, even if its value is empty.
func (d Data) KeyExists(key string) bool {
	_, found := d.Values[key]
	return found
}

// FileExists returns true if data.Files[key] exists. When parsing a request
// body, the key is considered to be in existence if it was provided in the
// request body, even if the file is empty.
func (d Data) FileExists(key string) bool {
	_, found := d.Files[key]
	return found
}

// GetInt returns the first element in data[key] converted to an int.
func (d Data) GetInt(key string) int {
	if !d.KeyExists(key) || len(d.Values[key]) == 0 {
		return 0
	}
	str := d.Get(key)
	if result, err := strconv.Atoi(str); err != nil {
		return 0
	} else {
		return result
	}
}

// GetFloat returns the first element in data[key] converted to a float.
func (d Data) GetFloat(key string) float64 {
	if !d.KeyExists(key) || len(d.Values[key]) == 0 {
		return 0.0
	}
	str := d.Get(key)
	if result, err := strconv.ParseFloat(str, 64); err != nil {
		return 0.0
	} else {
		return result
	}
}

// GetBool returns the first element in data[key] converted to a bool.
func (d Data) GetBool(key string) bool {
	if !d.KeyExists(key) || len(d.Values[key]) == 0 {
		return false
	}
	str := d.Get(key)
	if result, err := strconv.ParseBool(str); err != nil {
		return false
	} else {
		return result
	}
}

// GetUUID returns the first element in data[key] converted to a uuid.
func (d Data) GetUUID(key string) *uuid.UUID {
	if !d.KeyExists(key) || len(d.Values[key]) == 0 {
		return nil
	}
	str := d.Get(key)
	if result, err := uuid.Parse(str); err != nil {
		return nil
	} else {
		return &result
	}
}

// GetBytes returns the first element in data[key] converted to a slice of
// bytes.
func (d Data) GetBytes(key string) []byte {
	return []byte(d.Get(key))
}

// GetFileBytes returns the body of the file associated with key. If there is no
// file associated with key, it returns nil (not an error). It may return an
// error if there was a problem reading the file. If you need to know whether or
// not the file exists (i.e. whether it was provided in the request),
// use the FileExists method.
func (d Data) GetFileBytes(key string) ([]byte, error) {
	fileHeader, found := d.Files[key]
	if !found {
		return nil, nil
	} else {
		file, err := fileHeader.Open()
		if err != nil {
			return nil, err
		}
		return io.ReadAll(file)
	}
}

// GetStringsSplit returns the first element in data[key] split into a slice
// delimited by delim.
func (d Data) GetStringsSplit(key, delim string) []string {
	if !d.KeyExists(key) || len(d.Values[key]) == 0 {
		return nil
	}
	return strings.Split(d.Values[key][0], delim)
}

// BindJSON binds v to the json data in the request body. It calls
// json.Unmarshal and sets the value of v.
func (d Data) BindJSON(v any) error {
	if len(d.jsonBody) == 0 {
		return nil
	}
	return json.Unmarshal(d.jsonBody, v)
}

// GetMapFromJSON assumes that the first element in data[key] is a json string,
// attempts to unmarshal it into a map[string]any, and if successful, returns
// the result. If unmarshaling was not successful, returns an error.
func (d Data) GetMapFromJSON(key string) (map[string]any, error) {
	if !d.KeyExists(key) || len(d.Values[key]) == 0 {
		return nil, nil
	}
	result := map[string]any{}
	if err := json.Unmarshal([]byte(d.Get(key)), &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetSliceFromJSON assumes that the first element in data[key] is a json
// string, attempts to unmarshal it into a []any, and if successful, returns
// the result. If unmarshaling was not successful, returns an error.
func (d Data) GetSliceFromJSON(key string) ([]any, error) {
	if !d.KeyExists(key) || len(d.Values[key]) == 0 {
		return nil, nil
	}
	result := []any{}
	if err := json.Unmarshal([]byte(d.Get(key)), &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetAndUnmarshalJSON assumes that the first element in data[key] is a json
// string and attempts to unmarshal it into v. If unmarshaling was not
// successful, returns an error.
// v should be a pointer to some data structure.
func (d Data) GetAndUnmarshalJSON(key string, v any) error {
	return json.Unmarshal(d.GetBytes(key), v)
}
