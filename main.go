package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/alexproskurov/web-app/controllers"
	"github.com/alexproskurov/web-app/migrations"
	"github.com/alexproskurov/web-app/models"
	"github.com/alexproskurov/web-app/templates"
	"github.com/alexproskurov/web-app/views"
	"github.com/gorilla/csrf"
	"github.com/joho/godotenv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	// Setup the database.
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

	err = models.MigrateFS(db, migrations.FS, ".")
	if err != nil {
		panic(err)
	}

	// Setup services.
	userService := models.UserService{
		DB: db,
	}
	sessionService := models.SessionService{
		DB: db,
	}

	// Setup middleware.
	umw := controllers.UserMiddleware{
		SessionService: &sessionService,
	}

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

	// Setup controllers.
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

	// Setup router and routes.
	r := chi.NewRouter()
	r.Use(csrfMw)
	r.Use(umw.SetUser)
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

	r.Get("/signup", userC.New)
	r.Post("/users", userC.Create)
	r.Get("/signin", userC.SignIn)
	r.Post("/signin", userC.ProcessSignIn)
	r.Post("/signout", userC.ProcessSignOut)
	r.Route("/users/me", func(r chi.Router) {
		r.Use(umw.RequireUser)
		r.Get("/", userC.CurrentUser)
	})

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Page not found", http.StatusNotFound)
	})

	// Start the server.
	fmt.Println("Starting the server on :3000...")
	err = http.ListenAndServe(":3000", r)
	if err != nil {
		panic(err)
	}
}
