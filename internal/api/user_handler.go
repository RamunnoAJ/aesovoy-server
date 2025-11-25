package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"regexp"

	"github.com/RamunnoAJ/aesovoy-server/internal/store"
	"github.com/RamunnoAJ/aesovoy-server/internal/utils"
)

type registerUserRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type UserHandler struct {
	userStore store.UserStore
	logger    *slog.Logger
}

func NewUserHandler(userStore store.UserStore, logger *slog.Logger) *UserHandler {
	return &UserHandler{
		userStore: userStore,
		logger:    logger,
	}
}

func (h *UserHandler) validateRegisterRequest(req *registerUserRequest) error {
	if req.Username == "" {
		return errors.New("username is required")
	}

	if len(req.Username) > 50 {
		return errors.New("username cannot be greater than 50 characters")
	}

	if req.Email == "" {
		return errors.New("email is required")
	}

	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(req.Email) {
		return errors.New("invalid email format")
	}

	if req.Password == "" {
		return errors.New("password is required")
	}

	return nil
}

// HandleRegisterUser godoc
// @Summary      Creates a user
// @Description  Creates a new user with a username, email, and password
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        body  body      registerUserRequest  true  "User data"
// @Success      201   {object}  UserResponse
// @Failure      400   {object}  utils.HTTPError
// @Failure      500   {object}  utils.HTTPError
// @Router       /users [post]
func (h *UserHandler) HandleRegisterUser(w http.ResponseWriter, r *http.Request) {
	var req registerUserRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		h.logger.Error("decoding register request:", "error", err)
		utils.Error(w, http.StatusBadRequest, "invalid request payload")
		return
	}

	err = h.validateRegisterRequest(&req)
	if err != nil {
		utils.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	user := &store.User{
		Username: req.Username,
		Email:    req.Email,
	}

	// how do we deal with their passwords
	err = user.PasswordHash.Set(req.Password)
	if err != nil {
		h.logger.Error("hashing password", "error", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	err = h.userStore.CreateUser(user)
	if err != nil {
		h.logger.Error("registering user", "error", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	utils.OK(w, http.StatusCreated, utils.Envelope{"user": user}, "", nil)
}
