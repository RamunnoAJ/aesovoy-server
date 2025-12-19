# API Reference

Base URL: `/api/v1`

Authentication: Bearer Token required for most endpoints.

## Authentication & Users

- `POST /tokens/authentication` - Login (Get Bearer Token)
- `POST /users` - Register a new user

## Products & Inventory

- `GET /products` - List products
- `POST /products` - Create product
- `GET /products/{id}` - Get product details
- `PATCH /products/{id}` - Update product
- `DELETE /products/{id}` - Delete product
- `POST /products/{id}/ingredients` - Add ingredient to product recipe
- `PATCH /products/{id}/ingredients/{ingredientID}` - Update ingredient in recipe
- `DELETE /products/{id}/ingredients/{ingredientID}` - Remove ingredient from recipe

- `GET /categories` - List categories
- `POST /categories` - Create category
- `GET /categories/{id}` - Get category
- `PATCH /categories/{id}` - Update category
- `DELETE /categories/{id}` - Delete category
- `GET /categories/{id}/products` - Get products in category

- `GET /ingredients` - List ingredients
- `POST /ingredients` - Create ingredient
- `GET /ingredients/{id}` - Get ingredient
- `PATCH /ingredients/{id}` - Update ingredient
- `DELETE /ingredients/{id}` - Delete ingredient

## Local Stock & Sales

- `GET /local_stock` - List local stock
- `POST /local_stock` - Initialize stock for product
- `GET /local_stock/{product_id}` - Get stock for product
- `PATCH /local_stock/{product_id}/adjust` - Adjust stock quantity

- `GET /local_sales` - List local sales
- `POST /local_sales` - Create local sale (POS)
- `GET /local_sales/{id}` - Get sale details

## Clients & Orders

- `GET /clients` - List clients (searchable)
- `POST /clients` - Create client
- `GET /clients/{id}` - Get client
- `PATCH /clients/{id}` - Update client

- `GET /orders` - List orders (filterable)
- `POST /orders` - Create order
- `GET /orders/{id}` - Get order details
- `PATCH /orders/{id}/state` - Update order state

## Providers & Expenses

- `GET /providers` - List providers (searchable)
- `POST /providers` - Create provider
- `GET /providers/{id}` - Get provider
- `PATCH /providers/{id}` - Update provider

- `GET /payment_methods` - List payment methods
- `POST /payment_methods` - Create payment method
- `GET /payment_methods/{id}` - Get payment method
- `DELETE /payment_methods/{id}` - Delete payment method

- `GET /expenses` - List expenses
- `POST /expenses` - Create expense
- `GET /expenses/{id}` - Get expense
- `DELETE /expenses/{id}` - Delete expense

## Billing

- `GET /invoices` - List generated invoice files (JSON)

---
*For full details, schemas, and examples, please refer to the [Swagger Specification](../swagger/swagger.yaml) or the Swagger UI.*
