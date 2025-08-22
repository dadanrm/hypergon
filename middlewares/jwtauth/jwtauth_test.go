package jwtauth_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dadanrm/hypergon"
	"github.com/dadanrm/hypergon/middlewares/jwtauth"
	"github.com/golang-jwt/jwt/v5"
)

const (
	testSecret      = "my-secret-key"
	testWrongSecret = "another-secret"
)

func generateToken(claims jwt.Claims, secret string, method jwt.SigningMethod) (string, error) {
	token := jwt.NewWithClaims(method, claims)

	return token.SignedString([]byte(secret))
}

func TestNew_PanicWithoutSecret(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected New() to panic when WithSecretKey is not provided, but it did not")
		}
	}()

	jwtauth.New()
}

func TestMiddleware_SuccessfulAuthentication(t *testing.T) {
	claims := jwt.MapClaims{
		"user_id": 42,
		"exp":     time.Now().Add(time.Hour).Unix(),
	}

	tokenString, err := generateToken(claims, testSecret, jwt.SigningMethodHS256)
	if err != nil {
		t.Fatalf("Failed to generate test token: %v", err)
	}

	authMiddleware := jwtauth.New(jwtauth.WithSecretKey(testSecret))

	testHandler := hypergon.HandlerFunc(func(w http.ResponseWriter, r *http.Request) hypergon.HypergonError {
		retrievedClaims, ok := jwtauth.GetClaims(r.Context())

		if !ok {
			t.Error("Expected claims to be present in the context, but it did not")
		}

		if id := retrievedClaims["user_id"]; id != float64(42) {
			t.Errorf("Expected user_id to be 42, but got %v", id)
		}

		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	rr := httptest.NewRecorder()
	handlerChain := authMiddleware(testHandler)

	handlerChain(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, but got %d", http.StatusOK, rr.Code)
	}
}

func TestMiddleware_CustomErrorHandler_PropagatesError(t *testing.T) {
	propagatedError := hypergon.HttpError(http.StatusTeapot, "I am a teapot")
	errorHandlerCalled := false

	customHandler := func(w http.ResponseWriter, r *http.Request, err error) hypergon.HypergonError {
		errorHandlerCalled = true
		return propagatedError
	}

	authMiddleware := jwtauth.New(
		jwtauth.WithSecretKey(testSecret),
		jwtauth.WithErrorHandler(customHandler),
	)

	failingHandler := hypergon.HandlerFunc(func(w http.ResponseWriter, r *http.Request) hypergon.HypergonError {
		t.Fatal("the protected handler should not be called")
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rr := httptest.NewRecorder()

	// 3. Execute and check the returned error
	returnedErr := authMiddleware(failingHandler)(rr, req)

	// 4. Assert that the handler was called and propagated the correct error
	if !errorHandlerCalled {
		t.Error("Expected the custom error handler to be called, but it was not")
	}
	if returnedErr == nil {
		t.Fatal("Expected a non-nil error to be returned, but got nil")
	}
	if returnedErr != propagatedError {
		t.Errorf("Expected error '%v', but got '%v'", propagatedError, returnedErr)
	}

	// 5. Assert that the response was NOT written to, as the error was propagated
	if rr.Code != http.StatusOK { // httptest.ResponseRecorder defaults to 200
		t.Errorf("Expected response code to be the default %d, but got %d (handler wrote to response)", http.StatusOK, rr.Code)
	}
	if rr.Body.Len() > 0 {
		t.Errorf("Expected empty response body, but got '%s'", rr.Body.String())
	}
}
