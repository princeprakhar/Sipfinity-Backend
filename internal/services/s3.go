// services/s3.go
package services

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/google/uuid"
)

type S3Service struct {
	client     *s3.S3
	bucketName string
	region     string
}

func NewS3Service(region, bucketName string, accessKey, secretKey string) *S3Service {
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(region),
		Credentials: credentials.NewStaticCredentials(
			accessKey,
			secretKey,
			"",
		),
	}))

	return &S3Service{
		client:     s3.New(sess),
		bucketName: bucketName,
		region:     region,
	}
}

type UploadResult struct {
	Key         string
	URL         string
	FileName    string
	ContentType string
	Size        int64
}

func (s *S3Service) UploadImage(file multipart.File, header *multipart.FileHeader) (*UploadResult, error) {
	// Validate file type
	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		// Fallback to extension-based detection
		contentType = s.getContentTypeFromExtension(header.Filename)
	}
	
	if !s.isValidImageType(contentType) {
		return nil, fmt.Errorf("invalid file type: %s", contentType)
	}

	// Validate file size (e.g., max 10MB)
	const maxSize = 10 * 1024 * 1024 // 10MB
	if header.Size > maxSize {
		return nil, fmt.Errorf("file size too large: %d bytes (max: %d bytes)", header.Size, maxSize)
	}

	// Generate unique key with timestamp for better organization
	fileExt := filepath.Ext(header.Filename)
	timestamp := time.Now().Format("2006/01/02")
	key := fmt.Sprintf("products/images/%s/%s%s", timestamp, uuid.New().String(), fileExt)

	// Read file content
	buffer := bytes.NewBuffer(nil)
	if _, err := io.Copy(buffer, file); err != nil {
		return nil, fmt.Errorf("failed to read file: %v", err)
	}

	// Upload to S3
	_, err := s.client.PutObject(&s3.PutObjectInput{
		Bucket:      aws.String(s.bucketName),
		Key:         aws.String(key),
		Body:        bytes.NewReader(buffer.Bytes()),
		ContentType: aws.String(contentType),
		// ACL:         aws.String("public-read"),	
		CacheControl: aws.String("max-age=31536000"), // 1 year cache
	})

	if err != nil {
		return nil, fmt.Errorf("failed to upload to S3: %v", err)
	}

	// Generate S3 URL
	url := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", s.bucketName, s.region, key)

	return &UploadResult{
		Key:         key,
		URL:         url,
		FileName:    header.Filename,
		ContentType: contentType,
		Size:        header.Size,
	}, nil
}

func (s *S3Service) UploadMultipleImages(files []*multipart.FileHeader) ([]*UploadResult, error) {
	var results []*UploadResult
	var uploadErrors []string

	for i, fileHeader := range files {
		file, err := fileHeader.Open()
		if err != nil {
			uploadErrors = append(uploadErrors, fmt.Sprintf("file %d: failed to open - %v", i+1, err))
			continue
		}

		result, err := s.UploadImage(file, fileHeader)
		file.Close()
		
		if err != nil {
			uploadErrors = append(uploadErrors, fmt.Sprintf("file %d (%s): %v", i+1, fileHeader.Filename, err))
			continue
		}

		results = append(results, result)
	}

	if len(uploadErrors) > 0 {
		// If some uploads failed, clean up successful ones
		for _, result := range results {
			s.DeleteImage(result.Key)
		}
		return nil, fmt.Errorf("upload errors: %s", strings.Join(uploadErrors, "; "))
	}

	return results, nil
}

func (s *S3Service) DeleteImage(key string) error {
	if key == "" {
		return nil // Nothing to delete
	}

	_, err := s.client.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(key),
	})
	return err
}

func (s *S3Service) DeleteMultipleImages(keys []string) error {
	if len(keys) == 0 {
		return nil
	}

	var objects []*s3.ObjectIdentifier
	for _, key := range keys {
		if key != "" {
			objects = append(objects, &s3.ObjectIdentifier{Key: aws.String(key)})
		}
	}

	if len(objects) == 0 {
		return nil
	}

	_, err := s.client.DeleteObjects(&s3.DeleteObjectsInput{
		Bucket: aws.String(s.bucketName),
		Delete: &s3.Delete{
			Objects: objects,
			Quiet:   aws.Bool(true),
		},
	})
	return err
}

func (s *S3Service) isValidImageType(contentType string) bool {
	validTypes := []string{
		"image/jpeg",
		"image/jpg", 
		"image/png",
		"image/gif",
		"image/webp",
		"image/bmp",
		"image/tiff",
	}

	for _, validType := range validTypes {
		if strings.EqualFold(contentType, validType) {
			return true
		}
	}
	return false
}

func (s *S3Service) getContentTypeFromExtension(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".bmp":
		return "image/bmp"
	case ".tiff", ".tif":
		return "image/tiff"
	default:
		return "application/octet-stream"
	}
}