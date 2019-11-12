package main

import (
	"crypto/subtle"
	"fmt"
	"log"
	"net/http"
	"net/http/fcgi"
	"os"
	"runtime"

	"github.com/dgrijalva/jwt-go"
	"github.com/joho/godotenv"
)

var mySigningKey []byte
var appAddr string
var username string
var password string
var realm string
var credentialsFile string
var databaseURL string
var featuresURL string

func init() {
	var envFile string
	envFile = ".uberwald.env"

	err := godotenv.Load(envFile)

	if err != nil {
		log.Fatal("Error loading .env file")
	}

	runtime.GOMAXPROCS(runtime.NumCPU())
	appAddr = os.Getenv("APPADDR")                   // e.g. "APPADDR=0.0.0.0:3000"
	mySigningKey = []byte(os.Getenv("MYSIGNINGKEY")) // e.g. "MYSIGNINGKEY=XXXXXXXXXXXXXXXXXXXXXXXX"
	username = os.Getenv("USERNAME")
	password = os.Getenv("PASSWORD")
	realm = os.Getenv("REALM")
	credentialsFile = os.Getenv("CREDENTIALSFILE")
	databaseURL = os.Getenv("DATABASEURL")
	featuresURL = os.Getenv("FEATURESURL")
}

func isAuthorized(endpoint func(http.ResponseWriter, *http.Request)) http.HandlerFunc {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if r.Header["Token"] != nil {

			token, err := jwt.Parse(r.Header["Token"][0], func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("There was an error")
				}
				return mySigningKey, nil
			})

			if err != nil {
				fmt.Fprintf(w, err.Error())
			}

			if token.Valid {
				endpoint(w, r)
			}

		} else {
			http.Error(w, "Not Authorized", 401)
		}
	})
}

// BasicAuth wraps a handler requiring HTTP basic auth for it using the given
// username and password and the specified realm, which shouldn't contain quotes.
//
// Most web browser display a dialog with something like:
//
//    The website says: "<realm>"
//
// Which is really stupid so you may want to set the realm to a message rather than
// an actual realm.
func BasicAuth(handler http.HandlerFunc, username, password, realm string) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		user, pass, ok := r.BasicAuth()

		if !ok || subtle.ConstantTimeCompare([]byte(user), []byte(username)) != 1 || subtle.ConstantTimeCompare([]byte(pass), []byte(password)) != 1 {
			w.Header().Set("WWW-Authenticate", `Basic realm="`+realm+`"`)
			http.Error(w, "Unauthorized.", http.StatusUnauthorized)
			return
		}

		handler(w, r)
	}
}

func basicAuth(h http.Handler, username, password, realm string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()

		if !ok || subtle.ConstantTimeCompare([]byte(user), []byte(username)) != 1 || subtle.ConstantTimeCompare([]byte(pass), []byte(password)) != 1 {
			w.Header().Set("WWW-Authenticate", `Basic realm="`+realm+`"`)
			http.Error(w, "Unauthorized.", http.StatusUnauthorized)
			return
		}

		h.ServeHTTP(w, r)
	})
}

func main() {
	var err error

	mux := http.NewServeMux()

	if appAddr != "" {
		// Run as a local web server
		fs := http.FileServer(http.Dir("static"))
		mux.Handle("/urwaldpate/update", http.StripPrefix("/urwaldpate/update", basicAuth(fs, username, password, realm)))
		mux.HandleFunc("/urwaldpate/update/upload", BasicAuth(upload, username, password, realm))
		mux.HandleFunc("/urwaldpate/hektar", isAuthorized(hektar))
		err = http.ListenAndServe(appAddr, mux)

	} else {
		// Run as FCGI via standard I/O
		fs := http.FileServer(http.Dir("static"))
		mux.Handle("/fcgi-bin/uberwald/", http.StripPrefix("/fcgi-bin/uberwald", basicAuth(fs, username, password, realm)))
		mux.HandleFunc("/fcgi-bin/uberwald/upload", BasicAuth(upload, username, password, realm))
		mux.HandleFunc("/fcgi-bin/uberwald/hektar", isAuthorized(hektar))
		err = fcgi.Serve(nil, mux)
	}

	if err != nil {
		log.Fatal(err)
	}

}
