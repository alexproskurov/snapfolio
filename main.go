package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/alexproskurov/web-app/controllers"
	"github.com/alexproskurov/web-app/models"
	"github.com/alexproskurov/web-app/templates"
	"github.com/alexproskurov/web-app/views"
	"github.com/gorilla/csrf"
	"github.com/joho/godotenv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.Get("/", controllers.StaticHandler(views.Must(views.ParseFS(
		templates.FS,
		"tailwind.gohtml", "home.gohtml",
	))))

	r.Get("/contact", controllers.StaticHandler(views.Must(views.ParseFS(
		templates.FS,
		"tailwind.gohtml", "contact.gohtml",
	))))

	r.Get("/faq", controllers.FAQ(views.Must(views.ParseFS(
		templates.FS,
		"tailwind.gohtml", "faq.gohtml",
	))))

	cfg := models.DefaultPostgresConfig()
	db, err := models.Open(cfg)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		panic(err)
	}
	userService := models.UserService{
		DB: db,
	}

	sessionService := models.SessionService{
		DB: db,
	}

	userC := controllers.User{
		UserService:    &userService,
		SessionService: &sessionService,
	}
	userC.Templates.New = views.Must(views.ParseFS(
		templates.FS,
		"tailwind.gohtml", "signup.gohtml",
	))
	userC.Templates.SignIn = views.Must(views.ParseFS(
		templates.FS,
		"tailwind.gohtml", "signin.gohtml",
	))
	r.Get("/signup", userC.New)
	r.Post("/users", userC.Create)
	r.Get("/signin", userC.SignIn)
	r.Post("/signin", userC.ProcessSignIn)
	r.Get("/users/me", userC.CurrentUser)

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Page not found", http.StatusNotFound)
	})
	fmt.Println("Starting the server on :3000...")

	err = godotenv.Load()
	if err != nil {
		log.Fatalf("err loading: %v", err)
	}
	csrfKey := os.Getenv("CSRF_KEY")
	csrfMw := csrf.Protect(
		[]byte(csrfKey),
		// TODO: Change this before deploying.
		csrf.Secure(false),
	)

	err = http.ListenAndServe(":3000", csrfMw(r))
	if err != nil {
		panic(err)
	}
}
