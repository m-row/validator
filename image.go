package validator

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"mime"
	"mime/multipart"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/disintegration/imaging"
	"github.com/m-row/validator/interfaces"
)

func (v *Validator) AssignImage(
	key string,
	m interfaces.HasImage,
	required bool,
	allowedScopes ...string,
) error {
	v.SaveOldImgThumbDists(m)

	if !v.Data.FileExists(key) && required {
		v.Check(false, key, v.T.ValidateRequired())
	}

	if v.Data.FileExists(key) {
		v.Permit(key, allowedScopes)
		img := v.Data.GetFile(key)
		_, params, err := mime.ParseMediaType(
			img.Header.Get("Content-Disposition"),
		)
		if err != nil {
			return err
		}

		filename := params["filename"]
		imgName, ext, err := v.imageName(filename, m)
		if err != nil {
			return err
		}
		imgBytes, err := v.Data.GetFileBytes(key)
		if err != nil {
			return err
		}

		imgVal := filepath.Join("uploads", m.TableName(), imgName)
		thumbVal := filepath.Join(
			"uploads",
			m.TableName(),
			"thumbs",
			fmt.Sprintf("thumb_%s", imgName),
		)

		m.SetImg(&imgVal)
		m.SetThumb(&thumbVal)

		// public is a hidden path on live urls are in the format:
		// https://proj.com/uploads/banners/thumbs/thumb_banners_1637_9577.jpeg
		// thats why the database value is set without it,
		// but the OS path is full
		imgDist := v.GetRootPath(filepath.Join("public", imgVal))
		thumbDist := v.GetRootPath(filepath.Join("public", thumbVal))

		distpath := filepath.Join(
			"public",
			"uploads",
			m.TableName(),
			"thumbs",
		)
		_, err = os.Stat(distpath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				if err := os.MkdirAll(distpath, 0o750); err != nil {
					return err
				}
				msg := fmt.Sprintf(
					"Directory created successfully: %s",
					distpath,
				)
				log.Println(msg)
			}
		}

		// Create a new file in the uploads directory
		dist, err := os.Create(filepath.Clean(imgDist))
		if err != nil {
			return err
		}
		defer dist.Close()

		if _, err := dist.WriteString(string(imgBytes)); err != nil {
			return err
		}
		if ext != ".svg" {
			if err := v.generateThumb(img, thumbDist); err != nil {
				return err
			}
		} else {
			// if its an svg keep the same file bytes
			// and create a new file in the uploads thumb directory
			thumbdist, err := os.Create(filepath.Clean(thumbDist))
			if err != nil {
				return err
			}
			defer thumbdist.Close()

			if _, err := thumbdist.WriteString(string(imgBytes)); err != nil {
				return err
			}
		}
		v.newImg = imgDist
		v.newThumb = thumbDist
		v.DeleteOldPicture()
	}
	return nil
}

// imageName for uploaded files.
func (v *Validator) imageName(
	filename string,
	m interfaces.HasImage,
) (string, string, error) {
	// this slice must be sorted alphabetically
	mimetypes := []string{
		// ".jfif",
		".jpe",
		".jpeg",
		".jpg",
		".png",
		".svg",
		// ".webp",
	}
	ext := filepath.Ext(filename)
	if ok := slices.Contains(mimetypes, ext); !ok {
		mimeError := fmt.Errorf(
			"file extension not allowed: %s, allowed files are: %s",
			ext,
			strings.Join(mimetypes, ","),
		)
		return "", ext, mimeError
	}
	// Image data
	randomNum := rand.Int63n(1_000_000) //nolint:gosec // dw
	imgName := m.TableName() +
		"_" +
		strconv.FormatInt(time.Now().UnixNano(), 10) +
		"_" +
		strconv.FormatInt(randomNum, 10) +
		ext

	return imgName, ext, nil
}

// generateThumb for the form file provided in a post or put request.
func (v *Validator) generateThumb(
	file *multipart.FileHeader,
	dist string,
) error {
	imageFile, err := file.Open()
	if err != nil {
		return err
	}
	decodedImg, err := imaging.Decode(imageFile)
	if err != nil {
		return err
	}
	resizedThumb := imaging.Resize(decodedImg, 150, 0, imaging.Lanczos)

	f, err := os.Create(filepath.Clean(dist))
	if err != nil {
		return err
	}

	if err := imaging.Encode(f, resizedThumb, imaging.PNG); err != nil {
		if err := f.Close(); err != nil {
			return err
		}
		return err
	}
	return f.Close()
}

// deleteFile removes a single file provided dist string from system.
func (v *Validator) deleteFile(dist string) {
	if dist != "" && strings.Contains(dist, ".") {
		if err := os.Remove(dist); err != nil {
			// file is deleted
			err = fmt.Errorf("failed to delete file: %s, error: %w", dist, err)
			log.Println(err.Error())
		}
	}
}

// DeleteNewPicture removes a newly uploaded image and its thumb.
func (v *Validator) DeleteNewPicture() {
	if v.newImg != "" {
		v.deleteFile(v.newImg)
	}
	if v.newThumb != "" {
		v.deleteFile(v.newThumb)
	}
}

// SaveOldImgThumbDists sets old file path instead of url img,
// thumb values on validator.
func (v *Validator) SaveOldImgThumbDists(m interfaces.HasImage) {
	v.oldImg = m.GetImg()
	if v.oldImg != nil {
		imgNoDomain := strings.ReplaceAll(
			*v.oldImg,
			v.DOMAIN+"/",
			"",
		)
		oldImgDist := v.GetRootPath(filepath.Join("public", imgNoDomain))
		v.oldImg = &oldImgDist
		m.SetImg(&imgNoDomain)
	}

	v.oldThumb = m.GetThumb()
	if v.oldThumb != nil {
		thumbNoDomain := strings.ReplaceAll(
			*v.oldThumb,
			v.DOMAIN+"/",
			"",
		)
		oldThumbDist := v.GetRootPath(
			filepath.Join("public", thumbNoDomain),
		)
		v.oldThumb = &oldThumbDist
		m.SetThumb(&thumbNoDomain)
	}
}

// DeleteOldPicture removes an existing image and its thumb
// after successful update of new files.
func (v *Validator) DeleteOldPicture() {
	if v.oldImg != nil {
		v.deleteFile(*v.oldImg)
	}
	if v.oldThumb != nil {
		v.deleteFile(*v.oldThumb)
	}
}
