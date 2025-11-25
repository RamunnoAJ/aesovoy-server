package api

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/RamunnoAJ/aesovoy-server/internal/store"
	"github.com/RamunnoAJ/aesovoy-server/internal/utils"
	"github.com/go-chi/chi/v5"
)

type registerProductRequest struct {
	CategoryID        int64   `json:"category_id"`
	Name              string  `json:"name"`
	Description       string  `json:"description"`
	UnitPrice         float64 `json:"unit_price"`
	DistributionPrice float64 `json:"distribution_price"`
}

type ProductHandler struct {
	productStore store.ProductStore
	logger       *slog.Logger
}

func NewProductHandler(productStore store.ProductStore, logger *slog.Logger) *ProductHandler {
	return &ProductHandler{
		productStore: productStore,
		logger:       logger,
	}
}

func (h *ProductHandler) validateRegisterRequest(req *registerProductRequest) error {
	var errs utils.ValidationErrors
	if req.CategoryID == 0 {
		errs = append(errs, utils.FieldError{Field: "category_id", Message: "is required"})
	}
	if strings.TrimSpace(req.Name) == "" {
		errs = append(errs, utils.FieldError{Field: "name", Message: "is required"})
	}
	if req.UnitPrice <= 0 {
		errs = append(errs, utils.FieldError{Field: "unit_price", Message: "must be > 0"})
	}
	if req.DistributionPrice <= 0 {
		errs = append(errs, utils.FieldError{Field: "distribution_price", Message: "must be > 0"})
	}

	if len(errs) > 0 {
		return errs
	}
	return nil
}

func (h *ProductHandler) HandleRegisterProduct(w http.ResponseWriter, r *http.Request) {
	var req registerProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("decoding register product: %v", err)
		utils.Error(w, http.StatusBadRequest, "invalid request payload")
		return
	}
	if err := h.validateRegisterRequest(&req); err != nil {
		if ve, ok := err.(utils.ValidationErrors); ok {
			utils.Fail(w, http.StatusBadRequest, "validation failed", ve)
			return
		}
		utils.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	pr := &store.Product{
		CategoryID:        req.CategoryID,
		Name:              req.Name,
		Description:       req.Description,
		UnitPrice:         req.UnitPrice,
		DistributionPrice: req.DistributionPrice,
	}

	if err := h.productStore.CreateProduct(pr); err != nil {
		h.logger.Error("creating product: %v", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	utils.OK(w, http.StatusCreated, utils.Envelope{"product": pr}, "", nil)
}

func (h *ProductHandler) HandleUpdateProduct(w http.ResponseWriter, r *http.Request) {
	productID, err := utils.ReadIDParam(r)
	if err != nil {
		h.logger.Error("readIDParam: %v", err)
		utils.Error(w, http.StatusBadRequest, "invalid product id")
		return
	}

	pr, err := h.productStore.GetProductByID(productID)
	if err != nil {
		h.logger.Error("getProductByID: %v", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if pr == nil {
		utils.Error(w, http.StatusNotFound, "product not found")
		return
	}

	var req struct {
		CategoryID        *int64   `json:"category_id"`
		Name              *string  `json:"name"`
		Description       *string  `json:"description"`
		UnitPrice         *float64 `json:"unit_price"`
		DistributionPrice *float64 `json:"distribution_price"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("decoding update product: %v", err)
		utils.Error(w, http.StatusBadRequest, "invalid request payload")
		return
	}

	if req.CategoryID != nil {
		pr.CategoryID = *req.CategoryID
	}
	if req.Name != nil {
		pr.Name = *req.Name
	}
	if req.Description != nil {
		pr.Description = *req.Description
	}
	if req.UnitPrice != nil {
		pr.UnitPrice = *req.UnitPrice
	}
	if req.DistributionPrice != nil {
		pr.DistributionPrice = *req.DistributionPrice
	}

	if err := h.productStore.UpdateProduct(pr); err != nil {
		h.logger.Error("updating product: %v", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	utils.OK(w, http.StatusOK, utils.Envelope{"product": pr}, "", nil)
}

func (h *ProductHandler) HandleGetProductByID(w http.ResponseWriter, r *http.Request) {
	productID, err := utils.ReadIDParam(r)
	if err != nil {
		h.logger.Error("readIDParam: %v", err)
		utils.Error(w, http.StatusBadRequest, "invalid product id")
		return
	}

	pr, err := h.productStore.GetProductByID(productID)
	if err != nil {
		h.logger.Error("getProductByID: %v", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if pr == nil {
		utils.Error(w, http.StatusNotFound, "product not found")
		return
	}

	utils.OK(w, http.StatusOK, utils.Envelope{"product": pr}, "", nil)
}

func (h *ProductHandler) HandleGetProducts(w http.ResponseWriter, r *http.Request) {
	prs, err := h.productStore.GetAllProduct()
	if err != nil {
		h.logger.Error("getAllProducts: %v", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	utils.OK(w, http.StatusOK, utils.Envelope{"products": prs}, "", nil)
}

func (h *ProductHandler) HandleGetProductsByCategory(w http.ResponseWriter, r *http.Request) {
	categoryID, err := utils.ReadIDParam(r)
	if err != nil {
		h.logger.Error("readIDParam: %v", err)
		utils.Error(w, http.StatusBadRequest, "invalid category id")
		return
	}
	prs, err := h.productStore.GetProductsByCategoryID(categoryID)
	if err != nil {
		h.logger.Error("getProductsByCategoryID: %v", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	utils.OK(w, http.StatusOK, utils.Envelope{"products": prs}, "", nil)
}

func (h *ProductHandler) HandleDeleteProduct(w http.ResponseWriter, r *http.Request) {
	productID, err := utils.ReadIDParam(r)
	if err != nil {
		h.logger.Error("readIDParam: %v", err)
		utils.Error(w, http.StatusBadRequest, "invalid product id")
		return
	}

	err = h.productStore.DeleteProduct(productID)
	if err == sql.ErrNoRows {
		utils.Error(w, http.StatusNotFound, "product not found")
		return
	}
	if err != nil {
		h.logger.Error("deleting product: %v", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type productIngredientRequest struct {
	IngredientID int64   `json:"ingredient_id"`
	Quantity     float64 `json:"quantity"`
	Unit         string  `json:"unit"`
}

func (h *ProductHandler) HandleAddIngredientToProduct(w http.ResponseWriter, r *http.Request) {
	productID, err := strconv.ParseInt(chi.URLParam(r, "productID"), 10, 64)
	if err != nil {
		h.logger.Error("reading product id param: %v", err)
		utils.Error(w, http.StatusBadRequest, "invalid product id")
		return
	}

	var req productIngredientRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("decoding add ingredient request: %v", err)
		utils.Error(w, http.StatusBadRequest, "invalid request payload")
		return
	}

	pi, err := h.productStore.AddIngredientToProduct(productID, req.IngredientID, req.Quantity, req.Unit)
	if err != nil {
		h.logger.Error("adding ingredient to product: %v", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	utils.OK(w, http.StatusCreated, utils.Envelope{"product_ingredient": pi}, "", nil)
}

func (h *ProductHandler) HandleUpdateProductIngredient(w http.ResponseWriter, r *http.Request) {
	productID, err := strconv.ParseInt(chi.URLParam(r, "productID"), 10, 64)
	if err != nil {
		h.logger.Error("reading product id param: %v", err)
		utils.Error(w, http.StatusBadRequest, "invalid product id")
		return
	}
	ingredientID, err := strconv.ParseInt(chi.URLParam(r, "ingredientID"), 10, 64)
	if err != nil {
		h.logger.Error("reading ingredient id param: %v", err)
		utils.Error(w, http.StatusBadRequest, "invalid ingredient id")
		return
	}

	var req struct {
		Quantity float64 `json:"quantity"`
		Unit     string  `json:"unit"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("decoding update ingredient request: %v", err)
		utils.Error(w, http.StatusBadRequest, "invalid request payload")
		return
	}

	pi, err := h.productStore.UpdateProductIngredient(productID, ingredientID, req.Quantity, req.Unit)
	if err != nil {
		if err == sql.ErrNoRows {
			utils.Error(w, http.StatusNotFound, "product ingredient not found")
			return
		}
		h.logger.Error("updating product ingredient: %v", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	utils.OK(w, http.StatusOK, utils.Envelope{"product_ingredient": pi}, "", nil)
}

func (h *ProductHandler) HandleRemoveIngredientFromProduct(w http.ResponseWriter, r *http.Request) {
	productID, err := strconv.ParseInt(chi.URLParam(r, "productID"), 10, 64)
	if err != nil {
		h.logger.Error("reading product id param: %v", err)
		utils.Error(w, http.StatusBadRequest, "invalid product id")
		return
	}
	ingredientID, err := strconv.ParseInt(chi.URLParam(r, "ingredientID"), 10, 64)
	if err != nil {
		h.logger.Error("reading ingredient id param: %v", err)
		utils.Error(w, http.StatusBadRequest, "invalid ingredient id")
		return
	}

	err = h.productStore.RemoveIngredientFromProduct(productID, ingredientID)
	if err != nil {
		if err == sql.ErrNoRows {
			utils.Error(w, http.StatusNotFound, "product ingredient not found")
			return
		}
		h.logger.Error("removing product ingredient: %v", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
