package services

import (
	"context"
	"errors"
	"fmt"
	"ftrack/models"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"golang.org/x/image/draw"
)

type MediaService struct {
	uploadPath    string
	baseURL       string
	maxFileSize   int64
	allowedTypes  map[string]bool
	thumbnailSize int
}

type UploadedFile struct {
	URL          string
	ThumbnailURL string
	Size         int64
	Filename     string
	MimeType     string
	Duration     int
	Dimensions   *models.MediaDimensions
}

type CompressedMedia struct {
	URL  string
	Size int64
}

func NewMediaService(uploadPath, baseURL string) *MediaService {
	// Ensure upload directory exists
	os.MkdirAll(uploadPath, 0755)
	os.MkdirAll(filepath.Join(uploadPath, "thumbnails"), 0755)

	allowedTypes := map[string]bool{
		"image/jpeg":         true,
		"image/png":          true,
		"image/gif":          true,
		"image/webp":         true,
		"video/mp4":          true,
		"video/mpeg":         true,
		"video/quicktime":    true,
		"audio/mpeg":         true,
		"audio/wav":          true,
		"audio/ogg":          true,
		"application/pdf":    true,
		"application/msword": true,
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
		"text/plain": true,
	}

	return &MediaService{
		uploadPath:    uploadPath,
		baseURL:       baseURL,
		maxFileSize:   50 * 1024 * 1024, // 50MB
		allowedTypes:  allowedTypes,
		thumbnailSize: 300,
	}
}

func (ms *MediaService) UploadFile(ctx context.Context, file multipart.File, header *multipart.FileHeader, userID string) (*UploadedFile, error) {
	// Validate file type
	contentType := header.Header.Get("Content-Type")
	if !ms.allowedTypes[contentType] {
		return nil, errors.New("unsupported file type")
	}

	// Validate file size
	if header.Size > ms.maxFileSize {
		return nil, errors.New("file too large")
	}

	// Generate unique filename
	ext := filepath.Ext(header.Filename)
	filename := fmt.Sprintf("%s_%s%s", userID, uuid.New().String(), ext)
	filePath := filepath.Join(ms.uploadPath, filename)

	// Create the file
	dst, err := os.Create(filePath)
	if err != nil {
		logrus.Errorf("Failed to create file %s: %v", filePath, err)
		return nil, errors.New("failed to save file")
	}
	defer dst.Close()

	// Copy the uploaded file to destination
	_, err = io.Copy(dst, file)
	if err != nil {
		logrus.Errorf("Failed to copy file content: %v", err)
		os.Remove(filePath) // Clean up
		return nil, errors.New("failed to save file")
	}

	// Build file URL
	fileURL := fmt.Sprintf("%s/media/%s", ms.baseURL, filename)

	uploadedFile := &UploadedFile{
		URL:      fileURL,
		Size:     header.Size,
		Filename: header.Filename,
		MimeType: contentType,
	}

	// Generate thumbnail for images
	if strings.HasPrefix(contentType, "image/") {
		thumbnailURL, dimensions, err := ms.generateImageThumbnail(filePath, filename)
		if err != nil {
			logrus.Errorf("Failed to generate thumbnail: %v", err)
			// Continue without thumbnail
		} else {
			uploadedFile.ThumbnailURL = thumbnailURL
			uploadedFile.Dimensions = dimensions
		}
	}

	// Get video duration and dimensions if applicable
	if strings.HasPrefix(contentType, "video/") {
		duration, dimensions, err := ms.getVideoMetadata(filePath)
		if err != nil {
			logrus.Errorf("Failed to get video metadata: %v", err)
		} else {
			uploadedFile.Duration = duration
			uploadedFile.Dimensions = dimensions
		}

		// Generate video thumbnail
		thumbnailURL, err := ms.generateVideoThumbnail(filePath, filename)
		if err != nil {
			logrus.Errorf("Failed to generate video thumbnail: %v", err)
		} else {
			uploadedFile.ThumbnailURL = thumbnailURL
		}
	}

	return uploadedFile, nil
}

func (ms *MediaService) DeleteFile(ctx context.Context, fileURL string) error {
	// Extract filename from URL
	filename := filepath.Base(fileURL)
	filePath := filepath.Join(ms.uploadPath, filename)

	// Delete main file
	err := os.Remove(filePath)
	if err != nil && !os.IsNotExist(err) {
		logrus.Errorf("Failed to delete file %s: %v", filePath, err)
		return err
	}

	// Delete thumbnail if exists
	thumbnailPath := filepath.Join(ms.uploadPath, "thumbnails", "thumb_"+filename)
	err = os.Remove(thumbnailPath)
	if err != nil && !os.IsNotExist(err) {
		logrus.Errorf("Failed to delete thumbnail %s: %v", thumbnailPath, err)
	}

	return nil
}

func (ms *MediaService) DownloadFile(ctx context.Context, fileURL string) ([]byte, error) {
	// Extract filename from URL
	filename := filepath.Base(fileURL)
	filePath := filepath.Join(ms.uploadPath, filename)

	// Read file
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.New("file not found")
		}
		logrus.Errorf("Failed to read file %s: %v", filePath, err)
		return nil, errors.New("failed to read file")
	}

	return data, nil
}

func (ms *MediaService) CompressMedia(ctx context.Context, media *models.MessageMedia, quality int, maxSize int64) (*CompressedMedia, error) {
	// Extract filename from URL
	filename := filepath.Base(media.URL)
	filePath := filepath.Join(ms.uploadPath, filename)

	// Only compress images for now
	if !strings.HasPrefix(media.MimeType, "image/") {
		return nil, errors.New("compression not supported for this media type")
	}

	// Open source image
	srcFile, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer srcFile.Close()

	// Decode image
	img, format, err := image.Decode(srcFile)
	if err != nil {
		return nil, err
	}

	// Calculate new dimensions if needed
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Resize if too large
	if maxSize > 0 {
		currentSize := int64(width * height * 4) // Rough estimate
		if currentSize > maxSize {
			ratio := float64(maxSize) / float64(currentSize)
			newWidth := int(float64(width) * ratio)
			newHeight := int(float64(height) * ratio)

			// Create resized image
			resized := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))
			draw.CatmullRom.Scale(resized, resized.Bounds(), img, img.Bounds(), draw.Over, nil)
			img = resized
		}
	}

	// Generate compressed filename
	ext := filepath.Ext(filename)
	nameWithoutExt := strings.TrimSuffix(filename, ext)
	compressedFilename := fmt.Sprintf("%s_compressed%s", nameWithoutExt, ext)
	compressedPath := filepath.Join(ms.uploadPath, compressedFilename)

	// Create compressed file
	compressedFile, err := os.Create(compressedPath)
	if err != nil {
		return nil, err
	}
	defer compressedFile.Close()

	// Encode with compression
	switch format {
	case "jpeg":
		err = jpeg.Encode(compressedFile, img, &jpeg.Options{Quality: quality})
	case "png":
		err = png.Encode(compressedFile, img)
	default:
		return nil, errors.New("unsupported image format for compression")
	}

	if err != nil {
		os.Remove(compressedPath) // Clean up
		return nil, err
	}

	// Get file info
	fileInfo, err := compressedFile.Stat()
	if err != nil {
		return nil, err
	}

	compressedURL := fmt.Sprintf("%s/media/%s", ms.baseURL, compressedFilename)

	return &CompressedMedia{
		URL:  compressedURL,
		Size: fileInfo.Size(),
	}, nil
}

func (ms *MediaService) generateImageThumbnail(filePath, filename string) (string, *models.MediaDimensions, error) {
	// Open image file
	file, err := os.Open(filePath)
	if err != nil {
		return "", nil, err
	}
	defer file.Close()

	// Decode image
	img, _, err := image.Decode(file)
	if err != nil {
		return "", nil, err
	}

	// Get original dimensions
	bounds := img.Bounds()
	origWidth := bounds.Dx()
	origHeight := bounds.Dy()

	dimensions := &models.MediaDimensions{
		Width:  origWidth,
		Height: origHeight,
	}

	// Calculate thumbnail dimensions
	thumbWidth, thumbHeight := ms.calculateThumbnailSize(origWidth, origHeight)

	// Create thumbnail
	thumbnail := image.NewRGBA(image.Rect(0, 0, thumbWidth, thumbHeight))
	draw.CatmullRom.Scale(thumbnail, thumbnail.Bounds(), img, img.Bounds(), draw.Over, nil)

	// Save thumbnail
	ext := filepath.Ext(filename)
	nameWithoutExt := strings.TrimSuffix(filename, ext)
	thumbnailFilename := fmt.Sprintf("thumb_%s%s", nameWithoutExt, ext)
	thumbnailPath := filepath.Join(ms.uploadPath, "thumbnails", thumbnailFilename)

	thumbnailFile, err := os.Create(thumbnailPath)
	if err != nil {
		return "", dimensions, err
	}
	defer thumbnailFile.Close()

	// Encode thumbnail
	if strings.ToLower(ext) == ".png" {
		err = png.Encode(thumbnailFile, thumbnail)
	} else {
		err = jpeg.Encode(thumbnailFile, thumbnail, &jpeg.Options{Quality: 80})
	}

	if err != nil {
		os.Remove(thumbnailPath) // Clean up
		return "", dimensions, err
	}

	thumbnailURL := fmt.Sprintf("%s/media/thumbnails/%s", ms.baseURL, thumbnailFilename)
	return thumbnailURL, dimensions, nil
}

func (ms *MediaService) generateVideoThumbnail(filePath, filename string) (string, error) {
	// This is a placeholder for video thumbnail generation
	// In a real implementation, you would use FFmpeg or similar
	logrus.Infof("Video thumbnail generation not implemented for %s", filename)
	return "", errors.New("video thumbnail generation not implemented")
}

func (ms *MediaService) getVideoMetadata(filePath string) (int, *models.MediaDimensions, error) {
	// This is a placeholder for video metadata extraction
	// In a real implementation, you would use FFmpeg or similar
	logrus.Infof("Video metadata extraction not implemented for %s", filePath)
	return 0, nil, errors.New("video metadata extraction not implemented")
}

func (ms *MediaService) calculateThumbnailSize(origWidth, origHeight int) (int, int) {
	// Calculate proportional thumbnail size
	if origWidth > origHeight {
		// Landscape
		if origWidth > ms.thumbnailSize {
			ratio := float64(ms.thumbnailSize) / float64(origWidth)
			return ms.thumbnailSize, int(float64(origHeight) * ratio)
		}
	} else {
		// Portrait or square
		if origHeight > ms.thumbnailSize {
			ratio := float64(ms.thumbnailSize) / float64(origHeight)
			return int(float64(origWidth) * ratio), ms.thumbnailSize
		}
	}

	// Return original size if smaller than thumbnail size
	return origWidth, origHeight
}

func (ms *MediaService) GetFileInfo(ctx context.Context, fileURL string) (*models.FileInfo, error) {
	filename := filepath.Base(fileURL)
	filePath := filepath.Join(ms.uploadPath, filename)

	fileInfo, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.New("file not found")
		}
		return nil, err
	}

	return &models.FileInfo{
		Filename:  fileInfo.Name(),
		Size:      fileInfo.Size(),
		CreatedAt: fileInfo.ModTime(),
	}, nil
}

func (ms *MediaService) ValidateFile(header *multipart.FileHeader) error {
	// Check file type
	contentType := header.Header.Get("Content-Type")
	if !ms.allowedTypes[contentType] {
		return errors.New("unsupported file type")
	}

	// Check file size
	if header.Size > ms.maxFileSize {
		return errors.New("file too large")
	}

	// Check filename
	if header.Filename == "" {
		return errors.New("filename is required")
	}

	// Check for dangerous file extensions
	ext := strings.ToLower(filepath.Ext(header.Filename))
	dangerousExts := []string{".exe", ".bat", ".cmd", ".scr", ".vbs", ".js"}
	for _, dangExt := range dangerousExts {
		if ext == dangExt {
			return errors.New("dangerous file extension")
		}
	}

	return nil
}

func (ms *MediaService) CleanupOldFiles(ctx context.Context, olderThan time.Duration) error {
	cutoffTime := time.Now().Add(-olderThan)

	// Walk through upload directory
	err := filepath.Walk(ms.uploadPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue on error
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check if file is older than cutoff
		if info.ModTime().Before(cutoffTime) {
			logrus.Infof("Cleaning up old file: %s", path)
			os.Remove(path)
		}

		return nil
	})

	return err
}

func (ms *MediaService) GetStorageStats(ctx context.Context) (*models.StorageStats, error) {
	var totalSize int64
	var fileCount int64

	err := filepath.Walk(ms.uploadPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue on error
		}

		if !info.IsDir() {
			totalSize += info.Size()
			fileCount++
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &models.StorageStats{
		TotalSize: totalSize,
		FileCount: fileCount,
		UsedSpace: totalSize,
	}, nil
}
