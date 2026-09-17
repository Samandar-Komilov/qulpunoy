package models

import "errors"

var (
	ErrUserNotFound  = errors.New("User not found")
	ErrUsernameTaken = errors.New("Username is already taken")
	ErrInvalidInput  = errors.New("Invalid username or password")
	ErrUnauthorized  = errors.New("Unauthorized")
	ErrForbidden     = errors.New("Forbidden")

	ErrProductNotFound   = errors.New("Product not found")
	ErrInsufficientStock = errors.New("Insufficient stock for product")
	ErrEmptyProductName  = errors.New("Product name cannot be empty")
	ErrInvalidStock      = errors.New("Stock quantity cannot be negative")
	ErrInvalidPrice      = errors.New("Price must be >=0")

	ErrOrderNotFound            = errors.New("Order not found")
	ErrDuplicateOrder           = errors.New("Order with this idempotency key already exists")
	ErrInvalidOrderStatus       = errors.New("Invalid order status transition")
	ErrInvalidOrderItemQuantity = errors.New("Order item quantity must be >0")
	ErrMissingProductID         = errors.New("Order item must have a product ID")
	ErrOrderAlreadyCancelled    = errors.New("Order is already cancelled")
	ErrOrderAlreadyConfirmed    = errors.New("Order is already confirmed")
	ErrEmptyOrderItems          = errors.New("Order must contain at least one item")
	ErrMissingIdempotencyKey    = errors.New("Idempotency-key header is required")
)
