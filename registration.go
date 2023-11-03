package main

import (
	"database/sql"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"log"
	"net/http"
	"os"
	"strings"
)

func userManagement(w http.ResponseWriter, r *http.Request) {
	const function = "userManagement"

	var count int
	var options []string
	var adminRole bool
	var filename string

	session, err := store.Get(r, "user-session")
	if err != nil {
		log.Println(err)
		return
	}

	if !session.IsNew {
		sessionRole := session.Values["role"]
		if sessionRole == "admin" {
			adminRole = true
		} else {
			adminRole = false
		}
	} else {
		log.Println("No seesion found")
		return
	}

	result := pDB.QueryRow("select count(*) from users")
	if err := result.Scan(&count); err != nil {
		ReturnJSONError(w, function, err, http.StatusInternalServerError, true)
	}

	sOptions := ""
	name := ""
	role := ""
	qry := "select name, role from users where role = 'user' and name <> 'administrator' order by name"
	if adminRole {
		qry = "select name, role from users where name <> 'administrator' order by name"
	}
	if result, err := pDB.Query(qry); err != nil {
		ReturnJSONError(w, function, err, http.StatusInternalServerError, true)
		return
	} else {
		for result.Next() {
			if err = result.Scan(&name, &role); err != nil {
				ReturnJSONError(w, function, err, http.StatusInternalServerError, true)
			} else {
				options = append(options, `["`+name+`","`+role+`"]`)
			}
		}
		sOptions = strings.Join(options[:], ",")
	}

	if adminRole {
		filename = webFiles + "/registration.html"
	} else {
		filename = webFiles + "/userRegistration.html"
	}
	if fileContent, err := os.ReadFile(filename); err != nil {
		ReturnJSONError(w, function, err, http.StatusInternalServerError, true)
		return
	} else {

		if _, err := fmt.Fprint(w, strings.Replace(string(fileContent), "<!--map-->", sOptions, -1)); err != nil {
			ReturnJSONError(w, function, err, http.StatusInternalServerError, true)
		}
	}
}

// Create a struct that models the structure of a user, both in the request body, and in the DB
type Credentials struct {
	Password string `json:"password"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

func addUser(w http.ResponseWriter, r *http.Request) {
	// Parse and decode the request body into a new `Credentials` instance

	creds := &Credentials{}
	creds.Username = r.FormValue("user")
	creds.Password = r.FormValue("password")
	creds.Role = r.FormValue("role")
	// Salt and hash the password using the bcrypt algorithm
	// The second argument is the cost of hashing, which we arbitrarily set as 8 (this value can be more or less, depending on the computing power you wish to utilize)
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(creds.Password), 8)

	var rowCount int
	if err := pDB.QueryRow("select count(*) from users where name = ?", strings.ToLower(creds.Username)).Scan(&rowCount); err != nil {
		// If there is any issue with inserting into the database, return a 500 error
		ReturnJSONError(w, "Signup", err, http.StatusInternalServerError, true)
		return
	}
	var qry string
	if rowCount > 0 {
		qry = `update users set password = ?, role = ? where name = ?`
	} else {
		qry = `insert into users (password, role, name) values (?, ?, ?)`
	}
	// Next, insert the username, along with the hashed password into the database
	if _, err = pDB.Query(qry, string(hashedPassword), creds.Role, strings.ToLower(creds.Username)); err != nil {
		// If there is any issue with inserting into the database, return a 500 error
		ReturnJSONError(w, "Signup", err, http.StatusInternalServerError, true)
		return
	}
	if err := loadUserCreds(); err != nil {
		log.Println(err)
	}
	userManagement(w, r)
}

func deleteUser(w http.ResponseWriter, r *http.Request) {
	// Parse and decode the request body into a new `Credentials` instance
	var sessionName string
	const function = "DeleteUser"

	creds := &Credentials{}
	creds.Username = r.FormValue("user")

	session, err := store.Get(r, "user-session")
	if err != nil {
		log.Println(err)
		return
	}

	if !session.IsNew {
		sessionName = fmt.Sprint(session.Values["user"])
	} else {
		log.Println("No session found")
		return
	}

	if strings.ToLower(sessionName) == strings.ToLower(creds.Username) {
		ReturnJSONErrorString(w, function, "You cannot delete yourself", http.StatusBadRequest, true)
		return
	}
	log.Println("Delete user")

	// Next, delete the username from the database
	if _, err := pDB.Query("delete from users where name = ?", strings.ToLower(creds.Username)); err != nil {
		// If there is any issue with inserting into the database, return a 500 error
		ReturnJSONError(w, "Signup", err, http.StatusInternalServerError, true)
		return
	}
	if err := loadUserCreds(); err != nil {
		log.Println(err)
	}
	userManagement(w, r)
}

type CredType struct {
	password []byte
	role     string
}

var userCreds map[string]CredType

func loadUserCreds() error {
	// Parse and decode the request body into a new `Credentials` instance

	var (
		name string
		cred CredType
	)
	if rows, err := pDB.Query("select name, password, ifnull(role, 'user') from users"); err != nil {
		// If there is any issue with reading from the database, return a 500 error
		log.Println(err)
		return err
	} else {
		defer CloseResult(rows)
		userCreds = make(map[string]CredType, 0)
		for rows.Next() {
			if err := rows.Scan(&name, &cred.password, &cred.role); err != nil {
				log.Println(err)
				return err
			} else {
				userCreds[name] = cred
			}
		}
	}
	return nil
}

func CloseResult(rows *sql.Rows) {
	if err := rows.Close(); err != nil {
		log.Println(err)
	}
}

func Logout(w http.ResponseWriter, r *http.Request) {
	log.Println("Logging out...")
	// Get a session. Get() always returns a session, even if empty.
	session, err := store.Get(r, "user-session")
	if err != nil {
		ReturnJSONError(w, "Logout", err, http.StatusInternalServerError, true)
		return
	}

	session.Options.MaxAge = -1

	if err := store.Save(r, w, session); err != nil {
		log.Println(err)
	}
	http.ServeFile(w, r, webFiles+"/Logout.html")
}

/**
Authenticate checks the session cookie. If a matches with an active cookie cannot be found or there was no session cookie it returns (nil, StatusUnauthorised)
	Any errors are returned with an http status code
	If a matching session is found it is refreshed and Authenticate returns (nil, 0)
*/
func Authenticate(w http.ResponseWriter, r *http.Request) (error, int) {
	const UNAUTHORISED = "You are not authorised to access this site"

	//	log.Println("Checking authorisation")
	// Get the API key from the headers

	if len(userCreds) == 0 {
		if err := loadUserCreds(); err != nil {
			log.Println(err)
		}
	}

	//	log.Println("Get the session")
	// Get a session. Get() always returns a session, even if empty.
	session, err := store.Get(r, "user-session")
	if err != nil {
		return err, http.StatusInternalServerError
	}

	if !session.IsNew {
		sessionRole := session.Values["role"]
		if sessionRole == "admin" || sessionRole == "user" {
			// Valid session role so we can continue
			//		log.Println("Session OK - ", r.URL)
			return nil, 0
		}
	}

	key := r.Header.Get("Authorization")
	// If the authorization key is included and matches we are good to go.
	if (len(key) > 0) && (key == currentSettings.APIKey) {
		// API key matched so continue
		return nil, 0
	}

	//	log.Println("Checking for special URLs")
	if strings.HasSuffix(r.URL.Path, "/images/logo.png") ||
		strings.HasSuffix(r.URL.Path, "/Login.html") ||
		strings.HasSuffix(r.URL.Path, "/ping") ||
		strings.HasPrefix(r.URL.Path, "/ws") {
		return nil, 0
	} else {
		//		log.Println("Checking for action-login : ", r.RequestURI)
		if strings.HasSuffix(r.RequestURI, "action-login") {
			//			log.Println("Process login")
			if err := r.ParseForm(); err != nil {
				return err, http.StatusBadRequest
			}
			user := r.FormValue("user")
			password := r.FormValue("password")
			if err := bcrypt.CompareHashAndPassword([]byte(userCreds[user].password), []byte(password)); err == nil {
				// Set some session values.
				//				log.Println("Password matched so setting the session key")
				session.Values["role"] = userCreds[user].role
				session.Values["user"] = user
				// Save it before we write to the response/return from the handler.
				err = session.Save(r, w)
				if err != nil {
					log.Println(err)
				}
				//			log.Println("Session saved")
				return nil, 1 // Return err = nil but code = 1 to signify a new successful login
			} else {
				log.Println("Bad credentials")
				return fmt.Errorf(UNAUTHORISED), http.StatusUnauthorized
			}
		}
	}
	return fmt.Errorf(UNAUTHORISED), http.StatusUnauthorized
}
