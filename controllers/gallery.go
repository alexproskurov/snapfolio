package controllers

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"

	"github.com/alexproskurov/web-app/context"
	"github.com/alexproskurov/web-app/errors"
	"github.com/alexproskurov/web-app/models"
	"github.com/go-chi/chi/v5"
)

type Gallery struct {
	Templates struct {
		New   Template
		Show  Template
		Edit  Template
		Index Template
	}
	GalleryService *models.GalleryService
}

func (g Gallery) New(w http.ResponseWriter, r *http.Request) {
	var data struct {
		Title string
	}
	data.Title = r.FormValue("title")
	g.Templates.New.Execute(w, r, data)
}

func (g Gallery) Create(w http.ResponseWriter, r *http.Request) {
	var data struct {
		UserID int
		Title  string
	}
	data.UserID = context.User(r.Context()).ID
	data.Title = r.FormValue("title")

	gallery, err := g.GalleryService.Create(data.UserID, data.Title)
	if err != nil {
		g.Templates.New.Execute(w, r, data, err)
		return
	}

	editPath := fmt.Sprintf("/galleries/%d/edit", gallery.ID)
	http.Redirect(w, r, editPath, http.StatusFound)
}

func (g Gallery) Show(w http.ResponseWriter, r *http.Request) {
	gallery, err := g.getGalleryByID(w, r)
	if err != nil {
		return
	}

	type Image struct {
		GalleryID       int
		Filename        string
		FilenameEscaped string
	}
	var data struct {
		ID     int
		Title  string
		Images []Image
	}
	data.ID = gallery.ID
	data.Title = gallery.Title
	images, err := g.GalleryService.Images(gallery.ID)
	if err != nil {
		log.Println(err)
		http.Error(w, "Something went wrong.", http.StatusInternalServerError)
		return
	}
	for _, image := range images {
		data.Images = append(data.Images, Image{
			GalleryID:       image.GalleryID,
			Filename:        image.Filename,
			FilenameEscaped: url.PathEscape(image.Filename),
		})
	}

	g.Templates.Show.Execute(w, r, data)
}

func (g Gallery) Edit(w http.ResponseWriter, r *http.Request) {
	gallery, err := g.getGalleryByID(w, r, userMustOwnGallery)
	if err != nil {
		return
	}

	type Image struct {
		GalleryID       int
		Filename        string
		FilenameEscaped string
	}
	var data struct {
		ID     int
		Title  string
		Images []Image
	}
	data.ID = gallery.ID
	data.Title = gallery.Title
	images, err := g.GalleryService.Images(gallery.ID)
	if err != nil {
		log.Println(err)
		http.Error(w, "Something went wrong.", http.StatusInternalServerError)
		return
	}
	for _, image := range images {
		data.Images = append(data.Images, Image{
			GalleryID:       image.GalleryID,
			Filename:        image.Filename,
			FilenameEscaped: url.PathEscape(image.Filename),
		})
	}

	g.Templates.Edit.Execute(w, r, data)
}

func (g Gallery) Update(w http.ResponseWriter, r *http.Request) {
	gallery, err := g.getGalleryByID(w, r, userMustOwnGallery)
	if err != nil {
		return
	}

	gallery.Title = r.FormValue("title")
	err = g.GalleryService.Update(gallery)
	if err != nil {
		http.Error(w, "Something went wrong.", http.StatusInternalServerError)
		return
	}
	editPath := fmt.Sprintf("/galleries/%d/edit", gallery.ID)
	http.Redirect(w, r, editPath, http.StatusFound)
}

func (g Gallery) Index(w http.ResponseWriter, r *http.Request) {
	type Gallery struct {
		ID    int
		Title string
	}
	var data struct {
		Galleries []Gallery
	}

	userID := context.User(r.Context()).ID
	galleries, err := g.GalleryService.GetByUserID(userID)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			err = errors.Public(err, "No galleries available.")
			g.Templates.Index.Execute(w, r, data, err)
			return
		}
		http.Error(w, "Something went wrong.", http.StatusInternalServerError)
		return
	}

	data.Galleries = make([]Gallery, len(galleries))
	for i, gallery := range galleries {
		data.Galleries[i] = Gallery{
			ID:    gallery.ID,
			Title: gallery.Title,
		}
	}
	g.Templates.Index.Execute(w, r, data)
}

func (g Gallery) Delete(w http.ResponseWriter, r *http.Request) {
	gallery, err := g.getGalleryByID(w, r, userMustOwnGallery)
	if err != nil {
		return
	}

	var data struct {
		ID int
	}
	data.ID = gallery.ID
	err = g.GalleryService.Delete(data.ID)
	if err != nil {
		http.Error(w, "Something went wrong.", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/galleries", http.StatusFound)
}

func (g Gallery) Image(w http.ResponseWriter, r *http.Request) {
	filename := g.filename(w, r)
	galleryID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid ID.", http.StatusNotFound)
		return
	}
	image, err := g.GalleryService.Image(galleryID, filename)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			http.Error(w, "Image not found.", http.StatusNotFound)
			return
		}
		log.Println(err)
		http.Error(w, "Something went wrong.", http.StatusInternalServerError)
		return
	}

	http.ServeFile(w, r, image.Path)
}

func (g Gallery) UploadImage(w http.ResponseWriter, r *http.Request) {
	gallery, err := g.getGalleryByID(w, r, userMustOwnGallery)
	if err != nil {
		return
	}

	err = r.ParseMultipartForm(5 << 20) //5MB
	if err != nil {
		http.Error(w, "Something went wrong.", http.StatusInternalServerError)
		return
	}
	fileHeaders := r.MultipartForm.File["images"]
	for _, fh := range fileHeaders {
		file, err := fh.Open()
		if err != nil {
			http.Error(w, "Something went wrong.", http.StatusInternalServerError)
			return
		}
		defer file.Close()

		err = g.GalleryService.CreateImage(gallery.ID, fh.Filename, file)
		if err != nil {
			var fileErr models.FileError
			if errors.As(err, &fileErr) {
				msg := fmt.Sprintf("%v has an invalid content type or extension. "+
					"Only png, gif, jpg files can be uploaded.", fh.Filename)
				http.Error(w, msg, http.StatusBadRequest)
				return
			}
			http.Error(w, "Something went wrong.", http.StatusInternalServerError)
			return
		}
	}
	editPath := fmt.Sprintf("/galleries/%d/edit", gallery.ID)
	http.Redirect(w, r, editPath, http.StatusFound)
}

func (g Gallery) DeleteImage(w http.ResponseWriter, r *http.Request) {
	filename := g.filename(w, r)
	gallery, err := g.getGalleryByID(w, r, userMustOwnGallery)
	if err != nil {
		return
	}
	err = g.GalleryService.DeleteImage(gallery.ID, filename)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			http.Error(w, "Image not found.", http.StatusNotFound)
			return
		}
		log.Println(err)
		http.Error(w, "Something went wrong.", http.StatusInternalServerError)
		return
	}

	editPath := fmt.Sprintf("/galleries/%d/edit", gallery.ID)
	http.Redirect(w, r, editPath, http.StatusFound)
}

func (g Gallery) filename(w http.ResponseWriter, r *http.Request) string {
	filename := chi.URLParam(r, "filename")
	return filepath.Base(filename)
}

type galleryOpt func(http.ResponseWriter, *http.Request, *models.Gallery) error

func (g Gallery) getGalleryByID(w http.ResponseWriter, r *http.Request, opts ...galleryOpt) (*models.Gallery, error) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid ID.", http.StatusNotFound)
		return nil, err
	}
	gallery, err := g.GalleryService.GetByID(id)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			http.Error(w, "Gallery not found.", http.StatusNotFound)
			return nil, err
		}
		http.Error(w, "Something went wrong.", http.StatusInternalServerError)
		return nil, err
	}
	for _, opt := range opts {
		err = opt(w, r, gallery)
		if err != nil {
			return nil, err
		}
	}

	return gallery, nil
}

func userMustOwnGallery(w http.ResponseWriter, r *http.Request, gallery *models.Gallery) error {
	user := context.User(r.Context())
	if gallery.UserID != user.ID {
		http.Error(w, "You are not authorized to edit this gallery.", http.StatusForbidden)
		return fmt.Errorf("user does not have access to this gallery")
	}
	return nil
}
