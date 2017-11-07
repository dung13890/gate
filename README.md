# Gate
[![Build Status](https://travis-ci.org/hiendv/gate.svg?branch=master)](https://travis-ci.org/hiendv/gate) [![GoDoc](https://godoc.org/github.com/hiendv/gate?status.svg)](https://godoc.org/github.com/hiendv/gate) [![Go Report Card](https://goreportcard.com/badge/github.com/hiendv/gate)](https://goreportcard.com/report/github.com/hiendv/gate)
An authentication and RBAC authorization library using JWT for Go.

### Features
- Simple and well-tested API
- Exported flexible contracts
- Developer friendly
- Persistence free

### Supported authentication drivers
- Password-based authentication
- OAuth2 (coming soon)

### Installation
```bash
go get github.com/hiendv/gate
```

### Usage
Quick example to get a taste of Gate
```go

var auth gate.Auth
var user gate.User
var err error

// some codes go here

// Login using password-based authentication & Issue the JWT
user, err = auth.Login(map[string]string{"username": "username", "password": "password"})
jwt, err := auth.IssueJWT(user)

// Authenticate with a given JWT
user, err = auth.Authenticate("eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyIjp7ImlkIjoiaWQiLCJ1c2VybmFtZSI6InVzZXJuYW1lIiwicm9sZXMiOlsicm9sZSJdfSwiZXhwIjoxNjA1MDUyODAwLCJqdGkiOiJjbGFpbXMtaWQiLCJpYXQiOjE2MDUwNDkyMDB9.b0gxC2uZRek-SPwHSqyLOoW_DjSYroSivLqJG96Zxl0")
err = auth.Authorize(user, "action", "object")
```

You may want to check these examples and tests:
- Password-based authentication [examples]](https://godoc.org/github.com/hiendv/gate/password#pkg-examples) & [tests](password/password_test.go)

### Credits
Big thanks to [dgrijalva/jwt-go](https://github.com/dgrijalva/jwt-go) for the enormous help dealing with JWT works.