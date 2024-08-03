package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"
)

type APIServer struct {
	listenAddr string
	store      Storage
}

type apiFunc func(http.ResponseWriter, *http.Request) error

type APIError struct {
	Error string `json:"error"`
}

func WriteJSON(w http.ResponseWriter, status int, value any) error {
	w.Header().Add("content-type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(value)
}

func makeHTTPHandleFunc(f apiFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := f(w, r); err != nil {
			// handling the error
			WriteJSON(w, http.StatusBadRequest, APIError{Error: err.Error()})
		}
	}
}

func NewAPIServer(listenAddr string, store Storage) *APIServer {
	return &APIServer{
		listenAddr: listenAddr,
		store:      store,
	}
}

func (s *APIServer) Run() {
	router := mux.NewRouter()
	baseRouter := router.PathPrefix("/api/v1").Subrouter()

	authRouter := baseRouter.PathPrefix("/auth").Subrouter()
	accountRouter := baseRouter.PathPrefix("/account").Subrouter()

	authRouter.HandleFunc("/sign-up", makeHTTPHandleFunc(s.handleSignUp))
	authRouter.HandleFunc("/sign-in", makeHTTPHandleFunc(s.handleSignIn))

	accountRouter.HandleFunc("/transfer", WithJWTAuth(makeHTTPHandleFunc(s.handleTransfer), s.store))
	accountRouter.HandleFunc("/{id}", WithJWTAuth(makeHTTPHandleFunc(s.handleAccountID), s.store))
	accountRouter.HandleFunc("", makeHTTPHandleFunc(s.handleAccount))

	log.Println("Server listening on", s.listenAddr)
	http.ListenAndServe(s.listenAddr, router)
}

func (s *APIServer) handleSignUp(w http.ResponseWriter, r *http.Request) error {
	if r.Method != "POST" {
		return fmt.Errorf("METHOD NOT ALLOWED %s", r.Method)
	}

	signUpRequest := SignUpRequest{}
	if err := json.NewDecoder(r.Body).Decode(&signUpRequest); err != nil {
		return fmt.Errorf("first name, last name, email, password and confirm password is required")
	}

	if signUpRequest.Password != signUpRequest.ConfirmPassword {
		return fmt.Errorf("password and confirm password should matcn")
	}

	hashedPassword, err := hashPassword(signUpRequest.Password)
	if err != nil {
		return fmt.Errorf("unable to create account please try again")
	}

	account := NewAccount(
		signUpRequest.FirstName,
		signUpRequest.LastName,
		signUpRequest.Email,
		hashedPassword,
	)

	_, err = s.store.CreateAccount(account)

	if err != nil {
		return fmt.Errorf("unable to create account please try again")
	}

	return WriteJSON(w, http.StatusCreated, "Sign up successful!")
}

func (s *APIServer) handleSignIn(w http.ResponseWriter, r *http.Request) error {
	if r.Method != "POST" {
		return fmt.Errorf("METHOD NOT ALLOWED %s", r.Method)
	}

	signInReq := SignInRequest{}
	if err := json.NewDecoder(r.Body).Decode(&signInReq); err != nil {
		return fmt.Errorf("unable to signin %s", r.Method)
	}

	account, err := s.store.SignIn(signInReq.Email, signInReq.Password)

	if err != nil {
		return err
	}

	if ok := verifyPassword(signInReq.Password, account.Password); !ok {
		return WriteJSON(w, http.StatusOK, APIError{Error: "Invalid Credentials!"})
	}

	token, err := CreateJWTToken(account)
	if err != nil {
		return WriteJSON(w, http.StatusOK, APIError{Error: err.Error()})
	}

	w.Header().Add("x-jwt-token", token)

	return WriteJSON(w, http.StatusOK, "signed up successfully")
}

func (s *APIServer) handleAccount(w http.ResponseWriter, r *http.Request) error {
	if r.Method == "GET" {
		return s.handleGetAccount(w, r)
	}

	return fmt.Errorf("METHOD NOT ALLOWED %s", r.Method)
}

func (s *APIServer) handleAccountID(w http.ResponseWriter, r *http.Request) error {
	if r.Method == "GET" {
		return s.handleGetAccountById(w, r)
	}

	if r.Method == "DELETE" {
		return s.handleDeleteAccount(w, r)
	}

	return fmt.Errorf("METHOD NOT ALLOWED %s", r.Method)
}

func (s *APIServer) handleGetAccount(w http.ResponseWriter, _ *http.Request) error {
	accounts, err := s.store.GetAccounts()
	if err != nil {
		return fmt.Errorf("unable to get account")
	}
	return WriteJSON(w, http.StatusOK, accounts)
}

func (s *APIServer) handleGetAccountById(w http.ResponseWriter, r *http.Request) error {
	id, err := getIdFromReq(r)

	if err != nil {
		return err
	}

	account, err := s.store.GetAccountById(id)

	if err != nil {
		return fmt.Errorf("account with id %v not found", id)
	}

	return WriteJSON(w, http.StatusOK, account)
}

func (s *APIServer) handleDeleteAccount(w http.ResponseWriter, r *http.Request) error {
	id, err := getIdFromReq(r)
	if err != nil {
		return err
	}

	if err = s.store.DeleteAccount(id); err != nil {
		return fmt.Errorf("unable to delete account %v", id)
	}

	return WriteJSON(w, http.StatusOK, "account deleted")
}

func (s *APIServer) handleTransfer(w http.ResponseWriter, r *http.Request) error {

	if r.Method == "POST" {
		defer r.Body.Close()
		transferReq := &TransferRequest{}

		if err := json.NewDecoder(r.Body).Decode(&transferReq); err != nil {
			return err
		}

		return WriteJSON(w, http.StatusOK, transferReq)
	}

	return fmt.Errorf("METHOD NOT ALLOWED %s", r.Method)
}

func getIdFromReq(r *http.Request) (int, error) {
	id, err := strconv.Atoi(mux.Vars(r)["id"])

	if err != nil {
		return id, fmt.Errorf("provide a numeric value for id")
	}

	return id, nil
}

func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 8)
	return string(bytes), err
}

func verifyPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))

	return err == nil
}
