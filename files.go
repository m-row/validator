package validator

import (
	"crypto/md5"
	"errors"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/h2non/filetype"
)

type FileData struct {
	OriginalFileName,
	FileName,
	FilePath,
	FileType string
	FileSize     int
	FileCheckSum string
}

func CheckSumMD5(buf []byte, len_buf int) [16]byte {
	if len_buf != 0 {
		if len(buf) > len_buf {
			buf = buf[:len_buf]
		}
	}
	return md5.Sum(buf)
}

func (v *Validator) AssignFile(
	key string,
	fileName *string,
	required bool,
	allowedScopes ...string,
) (*FileData, error) {
	v.SaveOldFileDists(*fileName)

	var fileData FileData

	if !v.Data.FileExists(key) && required {
		v.Check(false, key, v.T.ValidateRequired())
		return nil, errors.New(v.T.ValidateRequired())
	}

	if v.Data.FileExists(key) {
		v.Permit(key, allowedScopes)
		f := v.Data.GetFile(key)
		_, params, err := mime.ParseMediaType(
			f.Header.Get("Content-Disposition"),
		)
		if err != nil {
			return nil, err
		}

		filenameParam := params["filename"]
		fileData.OriginalFileName = filenameParam
		fileName, err := v.fileName(filenameParam)
		if err != nil {
			return nil, err
		}

		fileData.FileName = fileName
		fileData.FileSize = int(f.Size)
		// fileData.FileType = f.Header.Get("Content-Type")

		fileBytes, err := v.Data.GetFileBytes(key)
		if err != nil {
			return nil, err
		}

		ft, err := filetype.Match(fileBytes)
		if err != nil {
			return nil, err
		}
		if !filetype.IsImage(fileBytes) {

			err := errors.New(v.T.FileIsNotAnImage())
			return nil, err
		}
		fileData.FileType = ft.MIME.Value

		// first 8 bytes to calculate file checksum, more takes performance
		fileChecksum := CheckSumMD5(fileBytes, 8192)
		fileData.FileCheckSum = fmt.Sprintf("%x", fileChecksum)

		filepathVal := filepath.Join("private", "files", fileName)
		filepathDist := v.GetRootPath(filepathVal)
		fileData.FilePath = filepathVal

		// Create a new file in the uploads directory
		dist, err := os.Create(filepath.Clean(filepathDist))
		if err != nil {
			return nil, err
		}
		defer dist.Close()

		if _, err := dist.WriteString(string(fileBytes)); err != nil {
			return nil, err
		}
		v.newFile = filepathDist
		v.DeleteOldFile()
	}
	return &fileData, nil
}

// DeleteOldFile removes an existing image and its thumb
// after successful update of new files.
func (v *Validator) DeleteOldFile() {
	if v.oldFile != nil {
		v.deleteFile(*v.oldFile)
	}
}

// imageName for uploaded files.
func (v *Validator) fileName(filename string) (string, error) {
	// this slice must be sorted alphabetically
	mimetypes := []string{
		// images
		".jfif",
		".jpe",
		".jpeg",
		".jpg",
		".png",
		".bmp",
		".webp",

		// documents
		".psd",
		".pdf",
		".doc",
		".docx",
		".xls",
		".xlsx",

		// archives
		".zip",
	}
	ext := filepath.Ext(filename)
	if ok := slices.Contains(mimetypes, ext); !ok {
		mimeError := fmt.Errorf(
			"file extension not allowed: %s, allowed files are: %s",
			ext,
			strings.Join(mimetypes, ","),
		)
		return "", mimeError
	}
	docName := fmt.Sprintf(
		"%s%s",
		uuid.NewString(),
		ext,
	)

	return docName, nil
}

// DeleteNewFile removes a newly uploaded file
func (v *Validator) DeleteNewFile() {
	if v.newFile != "" {
		v.deleteFile(v.newFile)
	}
}

// SaveOldFileDists sets old file path instead of url img,
// thumb values on validator.
func (v *Validator) SaveOldFileDists(filename string) {
	v.oldFile = &filename
	if v.oldFile != nil {
		fileNoDomain := strings.ReplaceAll(
			*v.oldFile,
			v.DOMAIN+"/",
			"",
		)
		oldFileDist := v.GetRootPath(
			filepath.Join("private", "files", fileNoDomain),
		)
		v.oldFile = &oldFileDist
	}
}
