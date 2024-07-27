package models

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
)

type Gallery struct {
	ID     int
	UserID int
	Title  string
}

type GalleryService struct {
	DB           *sql.DB
	ImageService *imageService
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

	err = os.RemoveAll(s.ImageService.galleryDir(id))
	if err != nil {
		return fmt.Errorf("delete gallery images: %w", err)
	}

	return nil
}

func (s *GalleryService) Images(galleryID int) ([]Image, error) {
	return s.ImageService.Images(galleryID)
}

func (s *GalleryService) Image(galleryID int, filename string) (Image, error) {
	return s.ImageService.Image(galleryID, filename)
}

func (s *GalleryService) CreateImage(galleryID int, filename string, contents io.ReadSeeker) error {
	return s.ImageService.CreateImage(galleryID, filename, contents)
}

func (s *GalleryService) DeleteImage(galleryID int, filename string) error {
	return s.ImageService.DeleteImage(galleryID, filename)
}
