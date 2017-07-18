package main

import (
	"fmt"
	"net/http"
	"sync"

	"crypto/subtle"

	uuid "github.com/satori/go.uuid"
)

const loginPage = "<html><head><title>Login</title></head><body><form action='login' method='post'><input type='password' name='password'><input type='submit'></form></body></html>"

var sessionStore map[string]Client
var storageMutex sync.RWMutex

//Client infomation
type Client struct {
	loggedIn bool
}

func main() {
	sessionStore = make(map[string]Client)

	http.Handle("/hello", helloWorldHandler{})
	http.HandleFunc("/login", handleLogin)
	http.Handle("/secureHello", authentication(helloWorldHandler{}))
	http.ListenAndServe(":3000", nil)
}

type helloWorldHandler struct {
}

func (h helloWorldHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Println(w, "Hello world")
}

type authenticationMiddleware struct {
	wrappedHandler http.Handler
}

func (h authenticationMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session")
	if err != nil {
		if err != http.ErrNoCookie {
			fmt.Fprint(w, err)
			return
		}
		err = nil
	}

	var present bool
	var client Client
	if cookie != nil {
		storageMutex.RLock()
		client, present = sessionStore[cookie.Value]
		storageMutex.RUnlock()
	} else {
		present = false
	}

	if present == false {
		cookie = &http.Cookie{
			Name:  "session",
			Value: uuid.NewV4().String(),
		}
		client = Client{false}
		storageMutex.Lock()
		sessionStore[cookie.Value] = client
		storageMutex.Unlock()
	}

	http.SetCookie(w, cookie)
	if client.loggedIn == false {
		fmt.Fprint(w, loginPage)
		return
	}
	if client.loggedIn == true {
		h.wrappedHandler.ServeHTTP(w, r)
		return
	}
}

func authentication(h http.Handler) authenticationMiddleware {
	return authenticationMiddleware{h}
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session")
	if err != nil {
		if err != http.ErrNoCookie {
			fmt.Fprint(w, err)
			return
		}
		err = nil
	}

	var present = false
	var client Client
	if cookie != nil {
		storageMutex.RLock()
		client, present = sessionStore[cookie.Value]
		storageMutex.RUnlock()
	}

	if !present {
		cookie = &http.Cookie{
			Name:  "session",
			Value: uuid.NewV4().String(),
		}
		client = Client{false}
		storageMutex.Lock()
		sessionStore[cookie.Value] = client
		storageMutex.Unlock()
	}
	http.SetCookie(w, cookie)
	err = r.ParseForm()
	if err != nil {
		fmt.Fprint(w, err)
		return
	}
	if subtle.ConstantTimeCompare(
		[]byte(r.FormValue("password")),
		[]byte("password123")) == 1 {
		//login user
		client.loggedIn = true
		fmt.Fprintln(w, "Thank you for logging in.")
		storageMutex.Lock()
		sessionStore[cookie.Value] = client
		storageMutex.Unlock()
	} else {
		fmt.Fprintln(w, "Wrong password")
	}
}
