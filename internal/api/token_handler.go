package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/RamunnoAJ/aesovoy-server/internal/store"
	"github.com/RamunnoAJ/aesovoy-server/internal/tokens"
	"github.com/RamunnoAJ/aesovoy-server/internal/utils"
)

type TokenHandler struct {
	tokenStore store.TokenStore
	userStore  store.UserStore
	logger     *slog.Logger
}

type createTokenRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func NewTokenHandler(tokenStore store.TokenStore, userStore store.UserStore, logger *slog.Logger) *TokenHandler {
	return &TokenHandler{
		tokenStore: tokenStore,
		userStore:  userStore,
		logger:     logger,
	}
}

// HandleCreateToken godoc
// @Summary      Creates an authentication token
// @Description  Creates a new authentication token for a user
// @Tags         tokens
// @Accept       json
// @Produce      json
// @Param        body  body      createTokenRequest  true  "User credentials"
// @Success      201   {object}  TokenResponse
// @Failure      400   {object}  utils.HTTPError
// @Failure      401   {object}  utils.HTTPError
// @Failure      500   {object}  utils.HTTPError
// @Router       /tokens/authentication [post]
func (h *TokenHandler) HandleCreateToken(w http.ResponseWriter, r *http.Request) {
	var req createTokenRequest
	err := json.NewDecoder(r.Body).Decode(&req)

	if err != nil {
		h.logger.Error("createTokenRequest", "error", err)
		utils.Error(w, http.StatusBadRequest, "invalid request payload")
		return
	}

	// lets get the user
	user, err := h.userStore.GetUserByUsername(req.Username)
	if err != nil || user == nil {
		h.logger.Error("GetUserByUsername", "error", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	passwordsDoMatch, err := user.PasswordHash.Matches(req.Password)
	if err != nil {
		h.logger.Error("PasswordHash.Mathes", "error", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	if !passwordsDoMatch {
		utils.Error(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	token, err := h.tokenStore.CreateNewToken(user.ID, 24*time.Hour, tokens.ScopeAuth)
	if err != nil {
		h.logger.Error("Creating Token", "error", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return

	}

	utils.OK(w, http.StatusCreated, utils.Envelope{"auth_token": token}, "", nil)
}
