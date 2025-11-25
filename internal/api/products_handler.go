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

// HandleRegisterProduct godoc
// @Summary      Creates a product
// @Description  Creates a new product with a name, a description, a category, and prices
// @Tags         products
// @Accept       json
// @Produce      json
// @Param        body  body      registerProductRequest  true  "Product data"
// @Success      201   {object}  ProductResponse
// @Failure      400   {object}  utils.HTTPError
// @Failure      500   {object}  utils.HTTPError
// @Security     BearerAuth
// @Router       /products [post]
func (h *ProductHandler) HandleRegisterProduct(w http.ResponseWriter, r *http.Request) {
	var req registerProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("decoding register product", "error", err)
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
		h.logger.Error("creating product", "error", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	utils.OK(w, http.StatusCreated, utils.Envelope{"product": pr}, "", nil)
}

// HandleUpdateProduct godoc
// @Summary      Updates a product
// @Description  Updates a product's details
// @Tags         products
// @Accept       json
// @Produce      json
// @Param        id    path      int                     true  "Product ID"
// @Param        body  body      registerProductRequest  true  "Product data"
// @Success      200   {object}  ProductResponse
// @Failure      400   {object}  utils.HTTPError
// @Failure      404   {object}  utils.HTTPError
// @Failure      500   {object}  utils.HTTPError
// @Security     BearerAuth
// @Router       /products/{id} [patch]
func (h *ProductHandler) HandleUpdateProduct(w http.ResponseWriter, r *http.Request) {
	productID, err := utils.ReadIDParam(r)
	if err != nil {
		h.logger.Error("readIDParam", "error", err)
		utils.Error(w, http.StatusBadRequest, "invalid product id")
		return
	}

	pr, err := h.productStore.GetProductByID(productID)
	if err != nil {
		h.logger.Error("getProductByID", "error", err)
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
		h.logger.Error("decoding update product", "error", err)
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
		h.logger.Error("updating product", "error", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	utils.OK(w, http.StatusOK, utils.Envelope{"product": pr}, "", nil)
}

// HandleGetProductByID godoc
// @Summary      Gets a product
// @Description  Responds with a single product with a given ID
// @Tags         products
// @Produce      json
// @Param        id   path      int      true  "Product ID"
// @Success      200  {object}  ProductResponse
// @Failure      400  {object}  utils.HTTPError
// @Failure      404  {object}  utils.HTTPError
// @Failure      500  {object}  utils.HTTPError
// @Security     BearerAuth
// @Router       /products/{id} [get]
func (h *ProductHandler) HandleGetProductByID(w http.ResponseWriter, r *http.Request) {
	productID, err := utils.ReadIDParam(r)
	if err != nil {
		h.logger.Error("readIDParam", "error", err)
		utils.Error(w, http.StatusBadRequest, "invalid product id")
		return
	}

	pr, err := h.productStore.GetProductByID(productID)
	if err != nil {
		h.logger.Error("getProductByID", "error", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if pr == nil {
		utils.Error(w, http.StatusNotFound, "product not found")
		return
	}

	utils.OK(w, http.StatusOK, utils.Envelope{"product": pr}, "", nil)
}

// HandleGetProducts godoc
// @Summary      Gets all products
// @Description  Responds with a list of all products
// @Tags         products
// @Produce      json
// @Success      200  {object}  ProductsResponse
// @Failure      500  {object}  utils.HTTPError
// @Security     BearerAuth
// @Router       /products [get]
func (h *ProductHandler) HandleGetProducts(w http.ResponseWriter, r *http.Request) {
	prs, err := h.productStore.GetAllProduct()
	if err != nil {
		h.logger.Error("getAllProducts", "error", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	utils.OK(w, http.StatusOK, utils.Envelope{"products": prs}, "", nil)
}

// HandleGetProductsByCategory godoc
// @Summary      Gets all products from a category
// @Description  Responds with a list of all products that belong to a given category
// @Tags         categories
// @Produce      json
// @Param        id   path      int      true  "Category ID"
// @Success      200  {object}  ProductsResponse
// @Failure      400  {object}  utils.HTTPError
// @Failure      500  {object}  utils.HTTPError
// @Security     BearerAuth
// @Router       /categories/{id}/products [get]
func (h *ProductHandler) HandleGetProductsByCategory(w http.ResponseWriter, r *http.Request) {
	categoryID, err := utils.ReadIDParam(r)
	if err != nil {
		h.logger.Error("readIDParam", "error", err)
		utils.Error(w, http.StatusBadRequest, "invalid category id")
		return
	}
	prs, err := h.productStore.GetProductsByCategoryID(categoryID)
	if err != nil {
		h.logger.Error("getProductsByCategoryID", "error", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	utils.OK(w, http.StatusOK, utils.Envelope{"products": prs}, "", nil)
}

// HandleDeleteProduct godoc
// @Summary      Deletes a product
// @Description  Deletes a product with a given ID
// @Tags         products
// @Param        id   path      int  true  "Product ID"
// @Success      204
// @Failure      400  {object}  utils.HTTPError
// @Failure      404  {object}  utils.HTTPError
// @Failure      500  {object}  utils.HTTPError
// @Security     BearerAuth
// @Router       /products/{id} [delete]
func (h *ProductHandler) HandleDeleteProduct(w http.ResponseWriter, r *http.Request) {
	productID, err := utils.ReadIDParam(r)
	if err != nil {
		h.logger.Error("readIDParam", "error", err)
		utils.Error(w, http.StatusBadRequest, "invalid product id")
		return
	}

	err = h.productStore.DeleteProduct(productID)
	if err == sql.ErrNoRows {
		utils.Error(w, http.StatusNotFound, "product not found")
		return
	}
	if err != nil {
		h.logger.Error("deleting product", "error", err)
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

// HandleAddIngredientToProduct godoc
// @Summary      Adds an ingredient to a product
// @Description  Adds a new ingredient to a product's recipe
// @Tags         products
// @Accept       json
// @Produce      json
// @Param        productID  path      int                       true  "Product ID"
// @Param        body       body      productIngredientRequest  true  "Ingredient data"
// @Success      201        {object}  ProductIngredientResponse
// @Failure      400        {object}  utils.HTTPError
// @Failure      500        {object}  utils.HTTPError
// @Security     BearerAuth
// @Router       /products/{productID}/ingredients [post]
func (h *ProductHandler) HandleAddIngredientToProduct(w http.ResponseWriter, r *http.Request) {
	productID, err := strconv.ParseInt(chi.URLParam(r, "productID"), 10, 64)
	if err != nil {
		h.logger.Error("reading product id param", "error", err)
		utils.Error(w, http.StatusBadRequest, "invalid product id")
		return
	}

	var req productIngredientRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("decoding add ingredient request", "error", err)
		utils.Error(w, http.StatusBadRequest, "invalid request payload")
		return
	}

	pi, err := h.productStore.AddIngredientToProduct(productID, req.IngredientID, req.Quantity, req.Unit)
	if err != nil {
		h.logger.Error("adding ingredient to product", "error", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	utils.OK(w, http.StatusCreated, utils.Envelope{"product_ingredient": pi}, "", nil)
}

// HandleUpdateProductIngredient godoc
// @Summary      Updates a product's ingredient
// @Description  Updates an ingredient's quantity or unit in a product's recipe
// @Tags         products
// @Accept       json
// @Produce      json
// @Param        productID     path      int     true  "Product ID"
// @Param        ingredientID  path      int     true  "Ingredient ID"
// @Param        body          body      object{quantity=float64,unit=string}  true  "Ingredient data"
// @Success      200           {object}  ProductIngredientResponse
// @Failure      400           {object}  utils.HTTPError
// @Failure      404           {object}  utils.HTTPError
// @Failure      500           {object}  utils.HTTPError
// @Security     BearerAuth
// @Router       /products/{productID}/ingredients/{ingredientID} [patch]
func (h *ProductHandler) HandleUpdateProductIngredient(w http.ResponseWriter, r *http.Request) {
	productID, err := strconv.ParseInt(chi.URLParam(r, "productID"), 10, 64)
	if err != nil {
		h.logger.Error("reading product id param", "error", err)
		utils.Error(w, http.StatusBadRequest, "invalid product id")
		return
	}
	ingredientID, err := strconv.ParseInt(chi.URLParam(r, "ingredientID"), 10, 64)
	if err != nil {
		h.logger.Error("reading ingredient id param", "error", err)
		utils.Error(w, http.StatusBadRequest, "invalid ingredient id")
		return
	}

	var req struct {
		Quantity float64 `json:"quantity"`
		Unit     string  `json:"unit"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("decoding update ingredient request", "error", err)
		utils.Error(w, http.StatusBadRequest, "invalid request payload")
		return
	}

	pi, err := h.productStore.UpdateProductIngredient(productID, ingredientID, req.Quantity, req.Unit)
	if err != nil {
		if err == sql.ErrNoRows {
			utils.Error(w, http.StatusNotFound, "product ingredient not found")
			return
		}
		h.logger.Error("updating product ingredient", "error", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	utils.OK(w, http.StatusOK, utils.Envelope{"product_ingredient": pi}, "", nil)
}

// HandleRemoveIngredientFromProduct godoc
// @Summary      Removes an ingredient from a product
// @Description  Removes an ingredient from a product's recipe
// @Tags         products
// @Param        productID     path      int  true  "Product ID"
// @Param        ingredientID  path      int  true  "Ingredient ID"
// @Success      204
// @Failure      400  {object}  utils.HTTPError
// @Failure      404  {object}  utils.HTTPError
// @Failure      500  {object}  utils.HTTPError
// @Security     BearerAuth
// @Router       /products/{productID}/ingredients/{ingredientID} [delete]
func (h *ProductHandler) HandleRemoveIngredientFromProduct(w http.ResponseWriter, r *http.Request) {
	productID, err := strconv.ParseInt(chi.URLParam(r, "productID"), 10, 64)
	if err != nil {
		h.logger.Error("reading product id param", "error", err)
		utils.Error(w, http.StatusBadRequest, "invalid product id")
		return
	}
	ingredientID, err := strconv.ParseInt(chi.URLParam(r, "ingredientID"), 10, 64)
	if err != nil {
		h.logger.Error("reading ingredient id param", "error", err)
		utils.Error(w, http.StatusBadRequest, "invalid ingredient id")
		return
	}

	err = h.productStore.RemoveIngredientFromProduct(productID, ingredientID)
	if err != nil {
		if err == sql.ErrNoRows {
			utils.Error(w, http.StatusNotFound, "product ingredient not found")
			return
		}
		h.logger.Error("removing product ingredient", "error", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
