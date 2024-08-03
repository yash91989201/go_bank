package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	_ "github.com/lib/pq"
)

type Storage interface {
	CreateAccount(*Account) (*Account, error)
	SignIn(email, password string) (*Account, error)
	GetAccounts() ([]*Account, error)
	GetAccountById(int) (*Account, error)
	UpdateAccount(*Account) error
	DeleteAccount(int) error
}

type PostgresStore struct {
	db *sql.DB
}

func newPostgresStore() (*PostgresStore, error) {
	connStr := "user=postgres dbname=postgres password=password sslmode=disable"
	db, err := sql.Open("postgres", connStr)

	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return &PostgresStore{
		db: db,
	}, nil
}

func (s *PostgresStore) Init() error {
	return s.CreateAccountTable()
}

func (s *PostgresStore) CreateAccountTable() error {
	query := `--sql
	CREATE TABLE IF NOT EXISTS account(
		created_at timestamp,
		id serial primary key,
		first_name varchar(48),
		last_name varchar(48),
		email varchar(64) UNIQUE,
		password varchar(256) NOT NULL ,
		account_number int UNIQUE,
		balance int
	);`

	_, err := s.db.Exec(query)

	return err
}

func (s *PostgresStore) CreateAccount(account *Account) (*Account, error) {
	query := `--sql
	INSERT INTO account(first_name, last_name, email, password, account_number, balance, created_at)
	values($1, $2, $3, $4, $5, $6, $7)
	RETURNING created_at, id, first_name, last_name, email, account_number, balance;
	`
	rows, err := s.db.Query(
		query,
		account.FirstName,
		account.LastName,
		account.Email,
		account.Password,
		account.BankNumber,
		account.Balance,
		account.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return rowToAccount(rows)
}

func (s *PostgresStore) SignIn(email, password string) (*Account, error) {
	query := `--sql
	SELECT * FROM account WHERE email=$1;
	`
	rows, err := s.db.Query(query, email)

	if err != nil {
		return nil, fmt.Errorf("Account doesnot exists")
	}

	return rowToAccount(rows)
}

func (s *PostgresStore) UpdateAccount(*Account) error {
	return nil
}

func (s *PostgresStore) DeleteAccount(id int) error {
	query := `--sql DELETE FROM account WHERE id=$1;`

	_, err := s.db.Query(query, id)

	if err != nil {
		return err
	}

	return nil
}

func permissionDenied(w http.ResponseWriter) {
	WriteJSON(w, http.StatusForbidden, APIError{Error: "Permission Denied"})
}

func WithJWTAuth(handler http.HandlerFunc, s Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tokenString := r.Header.Get("x-jwt-token")

		token, err := ValidateJWT(tokenString)
		fmt.Print(err)
		if err != nil {
			permissionDenied(w)
			return
		}

		claims := token.Claims.(jwt.MapClaims)
		userId := int(claims["id"].(float64))

		if _, err = s.GetAccountById(userId); err != nil {
			permissionDenied(w)
			return
		}

		handler(w, r)
	}
}

func CreateJWTToken(account *Account) (string, error) {
	secret := os.Getenv("JWT_SECRET")

	token := jwt.NewWithClaims(jwt.SigningMethodHS256,
		jwt.MapClaims{
			"id":  account.ID,
			"exp": time.Now().Add(time.Hour * 24).Unix(),
		})

	return token.SignedString([]byte(secret))
}

func ValidateJWT(tokenString string) (*jwt.Token, error) {
	secret := os.Getenv("JWT_SECRET")

	return jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
}

func (s *PostgresStore) GetAccounts() ([]*Account, error) {
	rows, err := s.db.Query(`SELECT * FROM account`)

	if err != nil {
		return nil, err
	}

	return rowsToAccounts(rows)
}

func (s *PostgresStore) GetAccountById(id int) (*Account, error) {
	query := `SELECT * FROM account WHERE id=$1`

	rows, err := s.db.Query(query, id)

	if err != nil {
		return nil, err
	}

	return rowToAccount(rows)
}

func rowToAccount(rows *sql.Rows) (*Account, error) {
	account := &Account{}

	rows.Next()
	err := rows.Scan(
		&account.CreatedAt,
		&account.ID,
		&account.FirstName,
		&account.LastName,
		&account.Email,
		&account.Password,
		&account.BankNumber,
		&account.Balance,
	)
	return account, err
}

func rowsToAccounts(rows *sql.Rows) ([]*Account, error) {

	accounts := []*Account{}
	for rows.Next() {
		account := &Account{}
		err := rows.Scan(
			&account.CreatedAt,
			&account.ID,
			&account.FirstName,
			&account.LastName,
			&account.BankNumber,
			&account.Balance,
		)

		if err != nil {
			return nil, err
		}

		accounts = append(accounts, account)
	}

	return accounts, nil
}
