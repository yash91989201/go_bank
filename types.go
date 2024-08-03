package main

import (
	"math/rand"
	"time"
)

type SignUpRequest struct {
	FirstName       string `json:"first_name"`
	LastName        string `json:"last_name"`
	Email           string `json:"email"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirm_password"`
}

type SignInRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type TransferRequest struct {
	ToAccount int `json:"to_account"`
	Amount    int `json:"amount"`
}

type Account struct {
	ID         int       `json:"id"`
	FirstName  string    `json:"first_name"`
	LastName   string    `json:"last_name"`
	Email      string    `json:"email"`
	Password   string    `json:"password"`
	BankNumber int64     `json:"bank_number"`
	Balance    int       `json:"balance"`
	CreatedAt  time.Time `json:"created_at"`
}

func NewAccount(FirstName, LastName, Email, Password string) *Account {
	return &Account{
		FirstName:  FirstName,
		LastName:   LastName,
		Email:      Email,
		Password:   Password,
		BankNumber: int64(rand.Intn(1000000)),
		Balance:    0,
		CreatedAt:  time.Now().UTC(),
	}
}
