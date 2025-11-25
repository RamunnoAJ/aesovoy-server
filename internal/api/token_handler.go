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

func (h *TokenHandler) HandleCreateToken(w http.ResponseWriter, r *http.Request) {
	var req createTokenRequest
	err := json.NewDecoder(r.Body).Decode(&req)

	if err != nil {
		h.logger.Error("createTokenRequest: %v", err)
		utils.Error(w, http.StatusBadRequest, "invalid request payload")
		return
	}

	// lets get the user
	user, err := h.userStore.GetUserByUsername(req.Username)
	if err != nil || user == nil {
		h.logger.Error("GetUserByUsername: %v", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	passwordsDoMatch, err := user.PasswordHash.Matches(req.Password)
	if err != nil {
		h.logger.Error("PasswordHash.Mathes %v", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	if !passwordsDoMatch {
		utils.Error(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	token, err := h.tokenStore.CreateNewToken(user.ID, 24*time.Hour, tokens.ScopeAuth)
	if err != nil {
		h.logger.Error("Creating Token %v", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return

	}

	utils.OK(w, http.StatusCreated, utils.Envelope{"auth_token": token}, "", nil)

}
