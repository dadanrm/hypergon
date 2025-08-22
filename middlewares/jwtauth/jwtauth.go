package jwtauth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/dadanrm/hypergon"
	"github.com/golang-jwt/jwt/v5"
)

type contextkey string

const claimsContextKey contextkey = "jwt_claims"

func New(configs ...configfunc) hypergon.Middleware {
	cfg := &jwtauthconfig{
		SecretKey:      os.Getenv("SECRET_KEY"), // we use the environment available variable for the default
		SigningMethod:  jwt.SigningMethodHS256,
		ErrorHandler:   defaulterrorhandler,
		TokenExtractor: FromAuthHeader,
	}

	for _, config := range configs {
		config(cfg)
	}

	if cfg.SecretKey == "" {
		panic("jwtauth middleware requires SecretKey")
	}

	secretKeyBytes := []byte(cfg.SecretKey)

	return func(hf hypergon.HandlerFunc) hypergon.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) hypergon.HypergonError {
			tokenString, err := cfg.TokenExtractor(r)
			if err != nil {
				return cfg.ErrorHandler(w, r, err)
			}

			token, err := jwt.Parse(tokenString, func(t *jwt.Token) (any, error) {
				if t.Method.Alg() != cfg.SigningMethod.Alg() {
					return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
				}

				return secretKeyBytes, nil
			})
			if err != nil {
				return cfg.ErrorHandler(w, r, err)
			}

			if !token.Valid {
				return cfg.ErrorHandler(w, r, errors.New("invalid token"))
			}

			claims, ok := token.Claims.(jwt.MapClaims)

			if !ok {
				return cfg.ErrorHandler(w, r, errors.New("failed to extract claims from token"))
			}

			ctx := context.WithValue(r.Context(), claimsContextKey, claims)
			r = r.WithContext(ctx)

			return hf(w, r)
		}
	}
}

type jwtauthconfig struct {
	SecretKey     string
	SigningMethod jwt.SigningMethod
	// Returns also the error that can be use for custom error handling
	ErrorHandler   func(w http.ResponseWriter, r *http.Request, err error) hypergon.HypergonError
	TokenExtractor func(r *http.Request) (string, error)
}

type configfunc func(*jwtauthconfig)

func WithSecretKey(key string) configfunc {
	return func(j *jwtauthconfig) {
		j.SecretKey = key
	}
}

func WithSingningMethod(method jwt.SigningMethod) configfunc {
	return func(j *jwtauthconfig) {
		j.SigningMethod = method
	}
}

func WithErrorHandler(handler func(w http.ResponseWriter, r *http.Request, err error) hypergon.HypergonError) configfunc {
	return func(j *jwtauthconfig) {
		j.ErrorHandler = handler
	}
}

func FromAuthHeader(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")

	if authHeader == "" {
		return "", errors.New("authorization header required")
	}

	parts := strings.Split(authHeader, " ")

	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return "", errors.New("invalid authorization header format")
	}

	return parts[1], nil
}

func GetClaims(ctx context.Context) (jwt.MapClaims, bool) {
	claims, ok := ctx.Value(claimsContextKey).(jwt.MapClaims)

	return claims, ok
}

func defaulterrorhandler(w http.ResponseWriter, r *http.Request, err error) hypergon.HypergonError {
	var errMsg string

	if errors.Is(err, jwt.ErrTokenExpired) {
		errMsg = "token has expired"
	} else if errors.Is(err, jwt.ErrTokenSignatureInvalid) {
		errMsg = "invalid token signature"
	} else {
		errMsg = "unauthorized"
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	fmt.Fprintf(w, `{"error": "%s"}`, errMsg)
	return nil
}
