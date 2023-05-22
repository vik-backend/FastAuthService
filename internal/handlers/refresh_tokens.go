package handlers

import (
	"AuthService/configs"
	"AuthService/internal/exceptions"
	"AuthService/internal/handlers/handlers_utils"
	"AuthService/internal/repositories/sessions_repo"
	"AuthService/internal/schemas"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"
)

// RefreshTokens is a handler function for token refresh requests.
// @Summary Refresh tokens
// @Description Use the refresh token to get a new access token and to set new refresh token in cookies.
// @Tags Auth
// @Accept  json
// @Produce  json
// @security JWTAuth
// @security ApiKeyAuth
// @Success 200 {object} schemas.TokenResponse
// @Failure 401 {object} schemas.ErrorResponse "Error returned when the provided auth data is invalid"
// @Failure 403 {object} schemas.ErrorResponse "Error returned when auth data was not provided"
// @Failure 500 {object} schemas.ErrorResponse "Internal server error"
// @Router /api/refresh [post]
func RefreshTokens(w http.ResponseWriter, r *http.Request) {
	existingSessionToken, userId, err := extractAndValidateTokens(r)
	if err != nil {
		handlers_utils.HandleException(w, err)
		return
	}

	accessToken, cookies, err := updateUserSessionAndGenerateTokens(r, existingSessionToken, userId)
	if err != nil {
		handlers_utils.HandleException(w, err)
		return
	}
	http.SetCookie(w, cookies)
	// Return the new access token
	err = handlers_utils.HandleJsonResponse(w, accessToken)
	if err != nil {
		handlers_utils.HandleException(w, fmt.Errorf("Error while handling JSON response: %v", err))
	}
}

// Extracts tokens, validates and returns necessary objects for new tokens creation
func extractAndValidateTokens(r *http.Request) (existingSessionToken string, userId int, err error) {
	log.Println("extractAndValidateTokens: validating tokens")
	// Extract the refresh token from the request cookies
	existingAccessToken, err := handlers_utils.ExtractJWT(r)
	if err != nil {
		return "", 0, err
	}
	// We dont validate existingAccessToken
	existingRefreshClaims, err := handlers_utils.ValidateRefreshTokenCookie(r, existingAccessToken)
	if err != nil {
		return "", 0, err
	}

	// Getting uid
	userId, err = strconv.Atoi((*existingRefreshClaims)["Id"].(string))
	log.Printf("extractAndValidateTokens: Got userId = %d", userId)
	if err != nil {
		return "", 0, err
	}
	sessionToken := (*existingRefreshClaims)["SessionToken"].(string)
	return sessionToken, userId, nil
}

func updateUserSessionAndGenerateTokens(r *http.Request, sessionToken string, userId int) (*schemas.TokenResponse, *http.Cookie, error) {
	// expiresAt declaration for session and for new refresh token
	expiresAt := time.Now().Add(time.Minute * time.Duration(configs.MainSettings.RefreshTokenLifeMinutes))

	// getting and updation of session directly if user is valid using custom sql request
	sessions, err := sessions_repo.OptimizedUpdateWithUserChecking(&expiresAt, sessionToken)
	if err != nil {
		switch err.(type) {
		case *exceptions.ErrNotFound:
			log.Println("Session for user was not found, it means filters are invalid, raising ErrUnauthorized")
			return nil, nil, &exceptions.ErrUnauthorized{Message: fmt.Sprint("Session for user was not found.")}
		default:
			return nil, nil, err
		}
	}
	if len(*sessions) != 1 {
		return nil, nil, fmt.Errorf("Found unexpected user session, please log in again.")
	}

	// getting device info
	deviceInfo := handlers_utils.GetDeviceInfo(r)
	// Generate a new access token
	accessToken, err := handlers_utils.GenerateAccessToken(userId, &deviceInfo)
	if err != nil {
		return nil, nil, err
	}
	// generate a new Refresh cookies using old session token
	cookies, err := handlers_utils.GenerateRefreshCookies(userId, accessToken.AccessToken, sessionToken, &expiresAt)
	if err != nil {
		return nil, nil, err
	}
	return accessToken, cookies, nil
}
