module auth-service

go 1.23.0

toolchain go1.23.6

require (
	github.com/golang-jwt/jwt/v5 v5.2.2
	github.com/gorilla/mux v1.8.1
	golang.org/x/oauth2 v0.29.0
)

require cloud.google.com/go/compute/metadata v0.3.0 // indirect
