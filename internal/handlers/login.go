package handlers

import (
	"AuthService/configs"
	"AuthService/internal/repositories/sessions_repo"
	"AuthService/internal/repositories/user_repo"
	"AuthService/internal/schemas"
	"AuthService/internal/utils"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// Login godoc
// @Summary Logs in a user
// @Description Authenticates a user using email and password, and generates a new JWT. Also sets refresh token in cookies.
// @Tags Auth
// @Accept  json
// @Produce  json
// @Param InputBody body schemas.LoginInput true "The email and password of the user"
// @Success 200 {object} schemas.TokenResponse "Returns a struct with the JWT and its expiration timestamp"
// @Failure 400 {object} schemas.ErrorResponse "Returns an error message if the request body cannot be parsed"
// @Failure 401 {object} schemas.ErrorResponse "Error returned when the provided auth data is invalid"
// @Failure 403 {object} schemas.ErrorResponse "Error returned when auth data was not provided"
// @Failure 500 {object} schemas.ErrorResponse "Returns an error message if there is a server-side issue"
// @Router /api/login [post]
func Login(w http.ResponseWriter, r *http.Request) {
	log.Println("Got request to login.")
	// Декодируем входные данные
	var input schemas.LoginInput
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		ErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Получаем пользователя из базы данных
	filters := make(map[string]interface{})
	filters["email"] = input.Email
	filters["deleted_at"] = nil
	user, err := user_repo.GetUser(&filters)
	if err != nil {
		HandleException(w, err)
		return
	}

	// Проверяем, соответствует ли предоставленный пароль хешу пароля
	isValid := utils.CheckPasswordHash(input.Password, user.Password)
	if !isValid {
		ErrorResponse(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	// getting device info
	deviceInfo := utils.GetDeviceInfo(r)
	log.Printf("deviceInfo \n	IP: %s\n	UserAgent: %s", deviceInfo.IPAddress, deviceInfo.UserAgent)

	// Генерируем токен доступа
	accessToken, err := utils.GenerateAccessToken(user, &deviceInfo)
	if err != nil {
		HandleException(w, err)
		return
	}

	sessionToken, err := utils.GenerateSessionToken(&deviceInfo, configs.MainSettings.SessionSecret)
	// generate the expiration date for both session and refreshToken
	expiresAt := time.Now().Add(time.Minute * time.Duration(configs.MainSettings.RefreshTokenLifeMinutes))
	sess, err := sessions_repo.CreateSession(user.ID, sessionToken, &expiresAt)
	if err != nil {
		HandleException(w, err)
		return
	}
	// Set Refresh cookies
	cookies, err := utils.GenerateRefreshCookies(user, accessToken.AccessToken, sessionToken, &sess.ExpiresAt)
	if err != nil {
		HandleException(w, err)
		return
	}
	http.SetCookie(w, &cookies)

	// Возвращаем токен в ответе
	err = HandleJsonResponse(w, accessToken)
	if err != nil {
		HandleException(w, fmt.Errorf("Error while handling JSON response: %v", err))
	}
}
