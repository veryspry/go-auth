package handlers

import (
	"encoding/json"
	"fmt"
	"go-chat/models"
	u "go-chat/utils"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/antonlindstrom/pgstore"
	"github.com/gorilla/securecookie"
)

// GetUserHandler - GET Route for user
func GetUserHandler(w http.ResponseWriter, r *http.Request) {
	// Read the email from query string
	email := r.URL.Query()["email"][0]
	// Lookup user record
	resp := models.GetUserByEmail(email)

	u.Respond(w, resp)
}

// GetUsersHandler gets all the Users in the system
func GetUsersHandler(w http.ResponseWriter, r *http.Request) {
	users := models.GetUsers()
	resp := u.Message(true, "Users found")
	resp["users"] = users
	u.Respond(w, resp)
}

// CreateUserHandler - POST route to create a user
func CreateUserHandler(w http.ResponseWriter, r *http.Request) {

	user := &models.User{}
	//decode the request body into struct
	err := json.NewDecoder(r.Body).Decode(user)
	if err != nil {
		u.Respond(w, u.Message(false, "Error creating user"))
		return
	}

	//Create account
	resp := user.Create()
	u.Respond(w, resp)
}

// Authenticate a user
func Authenticate(w http.ResponseWriter, r *http.Request) {

	user := &models.User{}

	//decode the request body into struct
	err := json.NewDecoder(r.Body).Decode(user)
	if err != nil {
		u.Respond(w, u.Message(false, "Invalid request"))
		return
	}

	resp := models.Login(user.Email, user.Password, w)

	// If the email address isn't found
	if resp["message"] == "Email address not found" {
		u.Respond(w, resp)
		return
	}

	// Type assert be a pointer to struct type models.User and then grab the token
	token := resp["user"].(*models.User).Token

	// Create a session and session cookie
	// Get token_secret
	tokenSecret := os.Getenv("token_secret")
	// Get db uri
	dbURI := models.GetDBURI()

	// Fetch new store.
	store, err := pgstore.NewPGStore(dbURI, []byte(tokenSecret))
	if err != nil {
		log.Fatalf(err.Error())
	}

	defer store.Close()

	// Run a background goroutine to clean up expired sessions from the database
	defer store.StopCleanup(store.Cleanup(time.Minute * 5))

	// Create a session.
	session, err := store.New(r, token)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		u.Respond(w, u.Message(false, "Internal server error"))
	}

	// Add a value
	session.Values["userEmail"] = user.Email
	fmt.Print(session.Options.Path)

	// Save
	if err = store.Save(r, w, session); err != nil {
		log.Fatalf("Error saving session: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		u.Respond(w, u.Message(false, "Internal server error"))
	}

	// if localhost, omit options.domain
	// Essentially, we need to do this because most browsers won't allow you to set a cookie if the domain field on cookie is present and you are on requeting from localhost
	referer := r.Referer()

	if strings.Contains(referer, "dev") {
		// Keep the session ID key in a cookie so it can be looked up in DB later.
		encoded, err := securecookie.EncodeMulti(session.Name(), session.ID, store.Codecs...)
		if err != nil {
			msg := u.Message(false, "Internal server error")
			w.WriteHeader(http.StatusInternalServerError)
			u.Respond(w, msg)
		}

		cookie := session.Name() + "=" + encoded

		w.Header().Set("Set-Cookie", cookie)
	}

	u.Respond(w, resp)
}
