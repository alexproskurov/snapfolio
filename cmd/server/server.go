package main

import (
	"fmt"
	"net/http"

	"github.com/alexproskurov/web-app/controllers"
	"github.com/alexproskurov/web-app/migrations"
	"github.com/alexproskurov/web-app/models"
	"github.com/alexproskurov/web-app/templates"
	"github.com/alexproskurov/web-app/views"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/gorilla/csrf"
	"github.com/spf13/viper"
)

type config struct {
	PSQL models.PostgresConfig `mapstructure:"psql"`
	SMTP models.SMTPConfig     `mapstructure:"smtp"`
	CSRF struct {
		Key    string
		Secure bool
	} `mapstructure:"csrf"`
	Server struct {
		Address string
	} `mapstructure:"server"`
}

func loadEnvConfig(path string) (*config, error) {
	v := viper.NewWithOptions(viper.KeyDelimiter("_"))
	v.AddConfigPath(path)
	v.SetConfigName(".env")
	v.SetConfigType("env")

	v.AutomaticEnv()

	err := v.ReadInConfig()
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg config
	err = v.Unmarshal(&cfg)
	if err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}

	return &cfg, nil
}

func main() {
	cfg, err := loadEnvConfig(".")
	if err != nil {
		panic(err)
	}

	// Setup the database.
	db, err := models.Open(cfg.PSQL)
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
	userService := &models.UserService{
		DB: db,
	}
	sessionService := &models.SessionService{
		DB: db,
	}
	pwResetService := &models.PasswordResetService{
		DB: db,
	}
	emailService := models.NewEmailService(cfg.SMTP)
	galleryService := &models.GalleryService{
		DB: db,
	}

	// Setup middleware.
	umw := controllers.UserMiddleware{
		SessionService: sessionService,
	}

	csrfMw := csrf.Protect(
		[]byte(cfg.CSRF.Key),
		csrf.Secure(cfg.CSRF.Secure),
		csrf.Path("/"),
	)

	// Setup controllers.
	userC := controllers.User{
		UserService:          userService,
		SessionService:       sessionService,
		PasswordResetService: pwResetService,
		EmailService:         emailService,
	}
	userC.Templates.New = views.Must(views.ParseFS(
		templates.FS,
		"tailwind.gohtml", "signup.gohtml",
	))
	userC.Templates.SignIn = views.Must(views.ParseFS(
		templates.FS,
		"tailwind.gohtml", "signin.gohtml",
	))
	userC.Templates.ForgotPassword = views.Must(views.ParseFS(
		templates.FS,
		"tailwind.gohtml", "forgot-pw.gohtml",
	))
	userC.Templates.CheckYourEmail = views.Must(views.ParseFS(
		templates.FS,
		"tailwind.gohtml", "check-your-email.gohtml",
	))
	userC.Templates.ResetPassword = views.Must(views.ParseFS(
		templates.FS,
		"tailwind.gohtml", "reset-pw.gohtml",
	))
	userC.Templates.ChangeEmail = views.Must(views.ParseFS(
		templates.FS,
		"tailwind.gohtml", "change-email.gohtml",
	))

	galleryC := controllers.Gallery{
		GalleryService: galleryService,
	}
	galleryC.Templates.New = views.Must(views.ParseFS(
		templates.FS,
		"tailwind.gohtml", "galleries/new.gohtml",
	))
	galleryC.Templates.Edit = views.Must(views.ParseFS(
		templates.FS,
		"tailwind.gohtml", "galleries/edit.gohtml",
	))
	galleryC.Templates.Index = views.Must(views.ParseFS(
		templates.FS,
		"tailwind.gohtml", "galleries/index.gohtml",
	))
	galleryC.Templates.Show = views.Must(views.ParseFS(
		templates.FS,
		"tailwind.gohtml", "galleries/show.gohtml",
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

	//users
	r.Get("/signup", userC.New)
	r.Get("/signin", userC.SignIn)
	r.Post("/signin", userC.ProcessSignIn)
	r.Post("/signout", userC.ProcessSignOut)
	r.Get("/forgot-pw", userC.ForgotPassword)
	r.Post("/forgot-pw", userC.ProcessForgotPassword)
	r.Get("/reset-pw", userC.ResetPassword)
	r.Post("/reset-pw", userC.ProcessResetPassword)
	r.Route("/users", func(r chi.Router) {
		r.Post("/", userC.Create)
		r.Group(func(r chi.Router) {
			r.Use(umw.RequireUser)
			r.Get("/me", userC.CurrentUser)
			r.Get("/edit", userC.ChangeEmail)
			r.Post("/edit", userC.ProcessChangeEmail)
		})
	})

	//galleries
	r.Route("/galleries", func(r chi.Router) {
		r.Get("/{id}", galleryC.Show)
		r.Get("/{id}/images/{filename}", galleryC.Image)
		r.Group(func(r chi.Router) {
			r.Use(umw.RequireUser)
			r.Get("/", galleryC.Index)
			r.Get("/new", galleryC.New)
			r.Post("/", galleryC.Create)
			r.Get("/{id}/edit", galleryC.Edit)
			r.Post("/{id}", galleryC.Update)
			r.Post("/{id}/delete", galleryC.Delete)
			r.Post("/{id}/images", galleryC.UploadImage)
			r.Post("/{id}/images/{filename}/delete", galleryC.DeleteImage)
		})
	})

	assetsHandler := http.FileServer(http.Dir("assets"))
	r.Get("/assets/*", http.StripPrefix("/assets", assetsHandler).ServeHTTP)

	//other
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Page not found", http.StatusNotFound)
	})

	// Start the server.
	fmt.Printf("Starting the server on %s...", cfg.Server.Address)
	err = http.ListenAndServe(cfg.Server.Address, r)
	if err != nil {
		panic(err)
	}
}
