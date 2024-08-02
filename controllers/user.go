package controllers

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/alexproskurov/snapfolio/context"
	"github.com/alexproskurov/snapfolio/errors"
	"github.com/alexproskurov/snapfolio/models"
)

type User struct {
	Templates struct {
		New            Template
		SignIn         Template
		ForgotPassword Template
		CheckYourEmail Template
		ResetPassword  Template
		ChangeEmail    Template
	}
	UserService          *models.UserService
	SessionService       *models.SessionService
	PasswordResetService *models.PasswordResetService
	EmailService         *models.EmailService
}

func (u User) New(w http.ResponseWriter, r *http.Request) {
	var data struct {
		Email string
	}
	data.Email = r.FormValue("email")
	u.Templates.New.Execute(w, r, data)
}

func (u User) Create(w http.ResponseWriter, r *http.Request) {
	var data struct {
		Email    string
		Password string
	}
	data.Email = r.FormValue("email")
	data.Password = r.FormValue("password")

	user, err := u.UserService.Create(data.Email, data.Password)
	if err != nil {
		if errors.Is(err, models.ErrEmailTaken) {
			err = errors.Public(err, "That email address is already associated with an account.")
		}
		u.Templates.SignIn.Execute(w, r, data, err)
		return
	}

	session, err := u.SessionService.Create(user.ID)
	if err != nil {
		err = errors.Public(err, "Unable to sign in. Please try again later.")
		u.Templates.SignIn.Execute(w, r, data, err)
		return
	}

	setCookie(w, CookieSession, session.Token)
	http.Redirect(w, r, "/galleries", http.StatusFound)
}

func (u User) SignIn(w http.ResponseWriter, r *http.Request) {
	var data struct {
		Email string
	}
	data.Email = r.FormValue("email")
	u.Templates.SignIn.Execute(w, r, data)
}

func (u User) ProcessSignIn(w http.ResponseWriter, r *http.Request) {
	var data struct {
		Email    string
		Password string
	}
	data.Email = r.FormValue("email")
	data.Password = r.FormValue("password")
	user, err := u.UserService.Authenticate(data.Email, data.Password)
	if err != nil {
		err = errors.Public(err, "Wrong email address or password. Try again or click Forgot password to reset it.")
		u.Templates.SignIn.Execute(w, r, data, err)
		return
	}

	session, err := u.SessionService.Create(user.ID)
	if err != nil {
		err = errors.Public(err, "Unable to sign in. Please try again later.")
		u.Templates.SignIn.Execute(w, r, data, err)
		return
	}

	setCookie(w, CookieSession, session.Token)
	http.Redirect(w, r, "/galleries", http.StatusFound)
}

func (u User) CurrentUser(w http.ResponseWriter, r *http.Request) {
	user := context.User(r.Context())
	fmt.Fprintf(w, "Current User: %s\n", user.Email)
}

func (u User) ProcessSignOut(w http.ResponseWriter, r *http.Request) {
	token, err := readCookie(r, CookieSession)
	if err != nil {
		http.Redirect(w, r, "/signin", http.StatusFound)
		return
	}

	err = u.SessionService.Delete(token)
	if err != nil {
		log.Println(err)
		http.Error(w, "Something went wrong.", http.StatusInternalServerError)
		return
	}

	deleteCookie(w, CookieSession)
	http.Redirect(w, r, "/signin", http.StatusFound)
}

func (u User) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var data struct {
		Email string
	}
	data.Email = r.FormValue("email")
	u.Templates.ForgotPassword.Execute(w, r, data)
}

func (u User) ProcessForgotPassword(w http.ResponseWriter, r *http.Request) {
	var data struct {
		Email string
	}
	data.Email = r.FormValue("email")

	pwReset, err := u.PasswordResetService.Create(data.Email)
	if err != nil {
		if errors.Is(err, models.ErrUserDoesNotExist) {
			err = errors.Public(err, "Couldn't find your SnapFolio Account")
			u.Templates.ForgotPassword.Execute(w, r, data, err)
			return
		}
		http.Error(w, "Something went wrong.", http.StatusInternalServerError)
		return
	}

	vals := url.Values{
		"token": {pwReset.Token},
	}
	resetURL := "https://www.snapfolio.com/reset-pw?" + vals.Encode()
	err = u.EmailService.ForgotPassword(data.Email, resetURL)
	if err != nil {
		err = errors.Public(err, "Something went wrong. Try again later.")
		u.Templates.ForgotPassword.Execute(w, r, data, err)
		return
	}
	// Don't render the reset token here! We need the user to confirm they have
	// access to the email account to verify their identity.
	u.Templates.CheckYourEmail.Execute(w, r, data)
}

func (u User) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var data struct {
		Token string
	}
	data.Token = r.FormValue("token")
	u.Templates.ResetPassword.Execute(w, r, data)
}

func (u User) ProcessResetPassword(w http.ResponseWriter, r *http.Request) {
	var data struct {
		Token    string
		Password string
	}
	data.Token = r.FormValue("token")
	data.Password = r.FormValue("password")

	user, err := u.PasswordResetService.Consume(data.Token)
	if err != nil {
		log.Println(err)
		if strings.Contains(err.Error(), "token expired:") {
			err = errors.Public(err, "Your session has expired. Please fill out the forgot password form again.")
			u.Templates.ResetPassword.Execute(w, r, data, err)
			return
		}
		http.Error(w, "Something went wrong.", http.StatusInternalServerError)
		return
	}

	err = u.UserService.UpdatePassword(user.ID, data.Password)
	if err != nil {
		http.Error(w, "Something went wrong.", http.StatusInternalServerError)
		return
	}

	// Sign the user is now that their password has been reset.
	// Any errors from this point onwards should redirect the user
	// to the sign in page.
	session, err := u.SessionService.Create(user.ID)
	if err != nil {
		err = errors.Public(err, "Unable to sign in. Please try again later.")
		u.Templates.SignIn.Execute(w, r, data, err)
		return
	}
	setCookie(w, CookieSession, session.Token)
	http.Redirect(w, r, "/users/me", http.StatusFound)
}

func (u User) ChangeEmail(w http.ResponseWriter, r *http.Request) {
	var data struct {
		Email string
	}
	data.Email = context.User(r.Context()).Email
	u.Templates.ChangeEmail.Execute(w, r, data)
}

func (u User) ProcessChangeEmail(w http.ResponseWriter, r *http.Request) {
	var data struct {
		Email string
	}
	data.Email = r.FormValue("email")
	user := context.User(r.Context())

	err := u.UserService.UpdateEmail(user.ID, data.Email)
	if err != nil {
		http.Error(w, "Something went wrong.", http.StatusInternalServerError)
		return
	}

	// Sign the user is now that their email has been updated.
	// Any errors from this point onwards should redirect the user
	// to the sign in page.
	session, err := u.SessionService.Create(user.ID)
	if err != nil {
		err = errors.Public(err, "Unable to sign in. Please try again later.")
		u.Templates.SignIn.Execute(w, r, data, err)
		return
	}
	setCookie(w, CookieSession, session.Token)
	http.Redirect(w, r, "/users/me", http.StatusFound)
}

type UserMiddleware struct {
	SessionService *models.SessionService
}

func (umw UserMiddleware) SetUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, err := readCookie(r, CookieSession)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		user, err := umw.SessionService.User(token)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		ctx := r.Context()
		ctx = context.WithUser(ctx, user)
		r = r.WithContext(ctx)
		next.ServeHTTP(w, r)
	})
}

func (umw UserMiddleware) RequireUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := context.User(r.Context())
		if user == nil {
			http.Redirect(w, r, "/signin", http.StatusFound)
			return
		}
		next.ServeHTTP(w, r)
	})
}
