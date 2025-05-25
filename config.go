package validator

import (
	"log"
	"net/http"
	"os"
	"path"

	"github.com/Masterminds/squirrel"
	"github.com/m-row/finder"
	"github.com/m-row/validator/interfaces"
	js "github.com/santhosh-tekuri/jsonschema/v5"
)

type Config struct {
	T       interfaces.Translation
	Conn    finder.Connection
	QB      *squirrel.StatementBuilderType
	Request *http.Request
	Scopes  []string
	Schema  *js.Schema
	RootDIR string
	DOMAIN  string
}

func (v *Validator) GetRootPath(dir string) string {
	ex, err := os.Executable()
	if err != nil {
		log.Fatalln(err)
	}
	return path.Join(path.Dir(ex), v.RootDIR, dir)
}
