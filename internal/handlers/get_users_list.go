package handlers

import (
	"AuthService/internal/handlers/handlers_utils"
	"AuthService/internal/repositories/user_repo"
	"fmt"
	"log"
	"net/http"
)

// GetManyUsers godoc
// @Summary Get many users
// @Description get many users based on pagination and sorting parameters
// @Tags Users
// @Accept json
// @Produce json
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Param sort query string false "Sorting (format: field[direction])"
// @Success 200 {array} models.User
// @Failure 401 {object} schemas.ErrorResponse "Error returned when the provided auth data is invalid"
// @Failure 403 {object} schemas.ErrorResponse "Error returned when auth data was not provided"
// @Failure 422 {object} schemas.ErrorResponse "Unprocessable entity"
// @Failure 500 {object} schemas.ErrorResponse "Internal server error"
// @Router /api/users [get]
func GetManyUsers(w http.ResponseWriter, r *http.Request) {
	log.Println("Got request to fetch many users.")
	params, err := handlers_utils.ExtractListParams(r)
	if err != nil {
		handlers_utils.HandleException(w, err)
		return
	}

	// Call GetManyUsers from the repo
	users, err := user_repo.GetManyUsers(*params)
	if err != nil {
		handlers_utils.HandleException(w, err)
		return
	}
	log.Printf("Successfully got users = %v", users)

	// Setting the status 200
	w.WriteHeader(http.StatusOK)

	// Prepare response
	err = handlers_utils.HandleJsonResponse(w, users)
	if err != nil {
		handlers_utils.HandleException(w, fmt.Errorf("Error while handling JSON response: %v", err))
	}
}
