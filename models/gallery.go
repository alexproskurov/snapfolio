package models

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type Image struct {
	GalleryID int
	Path      string
	Filename  string
}

type Gallery struct {
	ID     int
	UserID int
	Title  string
}

type GalleryService struct {
	DB *sql.DB

	// ImagesDir is used to tell the GalleryService where to store and locate
	// images. If not set, the GalleryService will default to using the "images"
	// directory.
	ImagesDir string
}

func (s *GalleryService) Create(userID int, title string) (*Gallery, error) {
	if userID < 0 {
		return nil, fmt.Errorf("create gallery: userID must be a positive number. userId = %d", userID)
	}
	if title == "" {
		return nil, fmt.Errorf("create gallery: empty title")
	}
	gallery := Gallery{
		UserID: userID,
		Title:  title,
	}

	row := s.DB.QueryRow(`
		INSERT INTO galleries (user_id, title)
		VALUES ($1, $2) RETURNING id;`, gallery.UserID, gallery.Title)
	err := row.Scan(&gallery.ID)
	if err != nil {
		return nil, fmt.Errorf("create gallery: %w", err)
	}

	return &gallery, nil
}

func (s *GalleryService) GetByID(id int) (*Gallery, error) {
	if id < 0 {
		return nil, fmt.Errorf("query gallery by id: id must be a positive number. id = %d", id)
	}
	gallery := Gallery{
		ID: id,
	}

	row := s.DB.QueryRow(`
		SELECT user_id, title 
		FROM galleries
		WHERE id = $1;`, gallery.ID)
	err := row.Scan(&gallery.UserID, &gallery.Title)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("query gallery by id: %w", err)
	}

	return &gallery, nil
}

func (s *GalleryService) GetByUserID(userID int) ([]Gallery, error) {
	if userID < 0 {
		return nil, fmt.Errorf("query galleries by user id: user id must be a positive number. user id = %d", userID)
	}

	rows, err := s.DB.Query(`
		SELECT id, title 
		FROM galleries
		WHERE user_id = $1;`, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("query galleries by user id: %w", err)
	}

	galleries := make([]Gallery, 0)
	for rows.Next() {
		gallery := Gallery{
			UserID: userID,
		}
		err = rows.Scan(&gallery.ID, &gallery.Title)
		if err != nil {
			return nil, fmt.Errorf("query galleries by user id: %w", err)
		}

		galleries = append(galleries, gallery)
	}
	if rows.Err() != nil {
		return nil, fmt.Errorf("query galleries by user id: %w", err)
	}

	return galleries, nil
}

func (s *GalleryService) Update(gallery *Gallery) error {
	if gallery.Title == "" {
		return fmt.Errorf("update gallery: empty title")
	}

	_, err := s.DB.Exec(`
		UPDATE galleries
		SET title = $2
		WHERE id = $1;`, gallery.ID, gallery.Title)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("update gallery: %w", err)
	}

	return nil
}

func (s *GalleryService) Delete(id int) error {
	if id < 0 {
		return fmt.Errorf("delete gallery: id must be a positive number. id = %d", id)
	}

	_, err := s.DB.Exec(`
		DELETE FROM galleries 
		WHERE id = $1;`, id)
	if err != nil {
		return fmt.Errorf("delete gallery: %w", err)
	}
	
	err = os.RemoveAll(s.galleryDir(id))
	if err != nil {
		return fmt.Errorf("delete gallery images: %w", err)
	}

	return nil
}

func (s *GalleryService) Images(galleryID int) ([]Image, error) {
	globPattern := filepath.Join(s.galleryDir(galleryID), "*")
	allFiles, err := filepath.Glob(globPattern)
	if err != nil {
		return nil, fmt.Errorf("retrieving gallery images: %w", err)
	}

	var images []Image
	for _, file := range allFiles {
		if hasExtension(file, s.extensions()) {
			images = append(images, Image{
				GalleryID: galleryID,
				Path:      file,
				Filename:  filepath.Base(file),
			})
		}
	}

	return images, nil
}

func (s *GalleryService) Image(galleryID int, filename string) (Image, error) {
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

func (s *GalleryService) CreateImage(galleryID int, filename string, contents io.ReadSeeker) error {
	err := checkContentType(contents, s.imageContentTypes())
	if err != nil {
		return fmt.Errorf("creating image %v: %w", filename, err)
	}
	err = checkExtension(filename, s.extensions())
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

func (s *GalleryService) DeleteImage(galleryID int, filename string) error {
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

func (s *GalleryService) extensions() []string {
	return []string{".png", ".jpg", ".jpeg", ".gif"}
}

func (s *GalleryService) imageContentTypes() []string {
	return []string{"image/png", "image/jpeg", "image/gif"}
}

func (s *GalleryService) galleryDir(id int) string {
	imagesDir := s.ImagesDir
	if imagesDir == "" {
		imagesDir = "images"
	}

	return filepath.Join(imagesDir, fmt.Sprintf("gallery-%d", id))
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
