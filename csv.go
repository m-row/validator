package validator

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"mime"
	"path/filepath"
	"slices"
	"strings"
)

var (
	ErrCsvInvalidHeaders = errors.New("invalid headers")
	ErrCsvNoRecords      = errors.New("no records")
)

func (v *Validator) ParseCSV(
	key string,
	csvHeader []string,
	required bool,
) ([][]string, error) {
	if len(csvHeader) == 0 {
		return nil, ErrCsvInvalidHeaders
	}
	fileExists := v.Data.FileExists(key)
	if !fileExists && required {
		v.Check(false, key, v.T.ValidateRequired())
	}

	if fileExists {
		csvFile := v.Data.GetFile(key)
		_, params, err := mime.ParseMediaType(
			csvFile.Header.Get("Content-Disposition"),
		)
		if err != nil {
			return nil, fmt.Errorf("ParseMediaType: %w", err)
		}

		fileName := params["filename"]
		ext := filepath.Ext(fileName)
		// this slice must be sorted alphabetically
		mimetypes := []string{
			".csv",
			".xlsx",
		}
		if ok := slices.Contains(mimetypes, ext); !ok {
			return nil, fmt.Errorf(
				"file extension not allowed: %s, allowed files are: %s",
				ext,
				strings.Join(mimetypes, ","),
			)
		}
		csvBytes, err := v.Data.GetFileBytes(key)
		if err != nil {
			return nil, fmt.Errorf("GetFileBytes: %w", err)
		}
		f := bytes.NewReader(csvBytes)
		r := csv.NewReader(f)
		r.Comma = ','
		records, err := r.ReadAll()
		if err != nil {
			return nil, fmt.Errorf("ReadAll: %w", err)
		}
		if len(records) == 0 {
			return nil, ErrCsvNoRecords
		}
		if len(records[0]) == 0 {
			return nil, ErrCsvNoRecords
		}
		firstRow := records[0]
		if len(firstRow) != len(csvHeader) {
			return nil, fmt.Errorf(
				"headers count provider: %d, needed: %d, should be only: %s",
				len(firstRow),
				len(csvHeader),
				strings.Join(csvHeader, ","),
			)
		}

		for i := range firstRow {
			if firstRow[i] != csvHeader[i] {
				v.Check(
					false,
					key,
					fmt.Sprintf(
						"header row[%d] should be: %s, found: %s",
						i,
						firstRow[i],
						csvHeader[i],
					),
				)
			}
		}
		return records[1:], nil
	}
	return nil, nil
}
