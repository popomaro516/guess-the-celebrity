package upload

import "errors"

var (
	ErrInvalidUpload        = errors.New("invalid upload")
	ErrUploadObjectNotFound = errors.New("upload object not found")
)
