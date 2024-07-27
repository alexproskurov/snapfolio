package main

import (
	"fmt"

	"github.com/alexproskurov/web-app/models"
)

func main() {
	gs := models.GalleryService{}
	fmt.Println(gs.Images(1))
}
