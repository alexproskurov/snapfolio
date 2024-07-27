package models

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type ImageStorager interface {
	Images(galleryID int) ([]Image, error)
	Image(galleryID int, filename string) (Image, error)
	CreateImage(galleryID int, filename string, contents io.ReadSeeker) error
	DeleteImage(galleryID int, filename string) error
	DeleteAllImages(galleryID int) error
}

var _ ImageStorager = (*DiskStorage)(nil)

type DiskStorage struct {
	// ImagesDir is used to tell the ImageService where to store and locate
	// images. If not set, the ImageService will default to using the "images"
	// directory.
	ImagesDir           string
	AllowedExtensions   []string
	AllowedContentTypes []string
}

type Image struct {
	GalleryID int
	Path      string
	Filename  string
}

func DefaultImageServiceConfig() *DiskStorage {
	return &DiskStorage{
		ImagesDir:           "images",
		AllowedExtensions:   []string{".png", ".jpg", ".jpeg", ".gif"},
		AllowedContentTypes: []string{"image/png", "image/jpeg", "image/gif"},
	}
}

func (s *DiskStorage) Images(galleryID int) ([]Image, error) {
	globPattern := filepath.Join(s.galleryDir(galleryID), "*")
	allFiles, err := filepath.Glob(globPattern)
	if err != nil {
		return nil, fmt.Errorf("retrieving gallery images: %w", err)
	}

	var images []Image
	for _, file := range allFiles {
		if hasExtension(file, s.AllowedExtensions) {
			images = append(images, Image{
				GalleryID: galleryID,
				Path:      file,
				Filename:  filepath.Base(file),
			})
		}
	}

	return images, nil
}

func (s *DiskStorage) Image(galleryID int, filename string) (Image, error) {
	imagePath := filepath.Join(s.galleryDir(galleryID), filename)
	_, err := os.Stat(imagePath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Image{}, ErrNotFound
		}
		return Image{}, fmt.Errorf("quering for image: %w", err)
	}

	return Image{
		GalleryID: galleryID,
		Path:      imagePath,
		Filename:  filename,
	}, nil
}

func (s *DiskStorage) CreateImage(galleryID int, filename string, contents io.ReadSeeker) error {
	err := checkContentType(contents, s.AllowedContentTypes)
	if err != nil {
		return fmt.Errorf("creating image %v: %w", filename, err)
	}
	err = checkExtension(filename, s.AllowedExtensions)
	if err != nil {
		return fmt.Errorf("creating image %v: %w", filename, err)
	}

	galleryDir := s.galleryDir(galleryID)
	err = os.MkdirAll(galleryDir, 0755)
	if err != nil {
		return fmt.Errorf("creating gallery-%d images directory: %w", galleryID, err)
	}

	imagePath := filepath.Join(galleryDir, filename)
	dst, err := os.Create(imagePath)
	if err != nil {
		return fmt.Errorf("creating image file: %w", err)
	}
	defer dst.Close()

	_, err = io.Copy(dst, contents)
	if err != nil {
		return fmt.Errorf("copying contents to image: %w", err)
	}
	return nil
}

func (s *DiskStorage) DeleteImage(galleryID int, filename string) error {
	image, err := s.Image(galleryID, filename)
	if err != nil {
		return fmt.Errorf("deleting image: %w", err)
	}
	err = os.Remove(image.Path)
	if err != nil {
		return fmt.Errorf("deleting image: %w", err)
	}

	return nil
}

func (d *DiskStorage) DeleteAllImages(galleryID int) error {
	err := os.RemoveAll(d.galleryDir(galleryID))
	if err != nil {
		return fmt.Errorf("delete all images related to gallery-%d: %w", galleryID, err)
	}
	return nil
}

func (s *DiskStorage) galleryDir(id int) string {
	return filepath.Join(s.ImagesDir, fmt.Sprintf("gallery-%d", id))
}

func hasExtension(file string, extensions []string) bool {
	for _, ext := range extensions {
		file = strings.ToLower(file)
		ext = strings.ToLower(ext)
		if filepath.Ext(file) == ext {
			return true
		}
	}
	return false
}
