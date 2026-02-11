package validation

import "time"

// Common request/response schemas for examples

// CreateUserRequest represents a user creation request
type CreateUserRequest struct {
	Name     string `json:"name" validate:"required,min=2,max=100" example:"John Doe" description:"Full name of the user"`
	Email    string `json:"email" validate:"required,email" example:"john@example.com" description:"Email address"`
	Password string `json:"password" validate:"required,password" example:"SecurePass123!" description:"Password with at least 8 characters"`
	Phone    string `json:"phone" validate:"phone" example:"+1234567890" description:"Phone number"`
	Age      int    `json:"age" validate:"min=18,max=120" example:"25" description:"Age of the user"`
}

// UpdateUserRequest represents a user update request
type UpdateUserRequest struct {
	Name  *string `json:"name,omitempty" validate:"omitempty,min=2,max=100" example:"Jane Doe" description:"Updated name"`
	Email *string `json:"email,omitempty" validate:"omitempty,email" example:"jane@example.com" description:"Updated email"`
	Phone *string `json:"phone,omitempty" validate:"omitempty,phone" example:"+1987654321" description:"Updated phone number"`
	Age   *int    `json:"age,omitempty" validate:"omitempty,min=18,max=120" example:"30" description:"Updated age"`
}

// UserResponse represents a user response
type UserResponse struct {
	ID              string    `json:"id" example:"550e8400-e29b-41d4-a716-446655440000" description:"Unique identifier"`
	Email           string    `json:"email" example:"john@example.com" description:"Email address"`
	IsEmailVerified bool      `json:"is_email_verified" example:"false" description:"Email verification status"`
	Is2FAEnabled    bool      `json:"is_2fa_enabled" example:"false" description:"2FA status"`
	CreatedAt       time.Time `json:"created_at" example:"2023-10-20T10:30:00Z" description:"Creation timestamp"`
}

// LoginRequest represents a login request
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email" example:"user@example.com" description:"Email address"`
	Password string `json:"password" validate:"required" example:"password123" description:"Password"`
}

// LoginResponse represents a login response
type LoginResponse struct {
	User         UserResponse `json:"user" description:"User information"`
	Tokens       TokenPair    `json:"tokens" description:"Access and refresh tokens"`
	Requires2FA  bool         `json:"requires_2fa" example:"false" description:"Whether 2FA is required"`
	RequiresOTP  bool         `json:"requires_otp" example:"false" description:"Whether OTP is required"`
}

// TokenPair represents access and refresh tokens
type TokenPair struct {
	AccessToken  string    `json:"access_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." description:"JWT access token"`
	RefreshToken string    `json:"refresh_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." description:"JWT refresh token"`
	ExpiresAt    time.Time `json:"expires_at" example:"2023-10-20T11:30:00Z" description:"Token expiration time"`
	TokenType    string    `json:"token_type" example:"Bearer" description:"Token type"`
}

// PaginationRequest represents pagination parameters
type PaginationRequest struct {
	Page     int `json:"page" validate:"min=1" example:"1" description:"Page number (starts from 1)"`
	PageSize int `json:"page_size" validate:"min=1,max=100" example:"10" description:"Number of items per page"`
}

// PaginationResponse represents pagination metadata
type PaginationResponse struct {
	Page       int   `json:"page" example:"1" description:"Current page number"`
	PageSize   int   `json:"page_size" example:"10" description:"Items per page"`
	Total      int64 `json:"total" example:"100" description:"Total number of items"`
	TotalPages int   `json:"total_pages" example:"10" description:"Total number of pages"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error" example:"validation_error" description:"Error type"`
	Message string `json:"message" example:"Invalid input data" description:"Error message"`
	Details string `json:"details,omitempty" example:"Email is required" description:"Additional error details"`
}

// SuccessResponse represents a success response
type SuccessResponse struct {
	Success bool        `json:"success" example:"true" description:"Success status"`
	Message string      `json:"message" example:"Operation completed successfully" description:"Success message"`
	Data    interface{} `json:"data,omitempty" description:"Response data"`
}

// CreateProductRequest represents a product creation request
type CreateProductRequest struct {
	Name        string  `json:"name" validate:"required,min=2,max=200" example:"iPhone 15" description:"Product name"`
	Description string  `json:"description" validate:"max=1000" example:"Latest iPhone model" description:"Product description"`
	Price       float64 `json:"price" validate:"required,min=0" example:"999.99" description:"Product price"`
	SKU         string  `json:"sku" validate:"required,alphanum" example:"IPH15-128-BLK" description:"Product SKU"`
	Category    string  `json:"category" validate:"required" example:"Electronics" description:"Product category"`
	InStock     bool    `json:"in_stock" example:"true" description:"Stock availability"`
}

// ProductResponse represents a product response
type ProductResponse struct {
	ID          string    `json:"id" example:"550e8400-e29b-41d4-a716-446655440000" description:"Unique identifier"`
	Name        string    `json:"name" example:"iPhone 15" description:"Product name"`
	Description string    `json:"description" example:"Latest iPhone model" description:"Product description"`
	Price       float64   `json:"price" example:"999.99" description:"Product price"`
	SKU         string    `json:"sku" example:"IPH15-128-BLK" description:"Product SKU"`
	Category    string    `json:"category" example:"Electronics" description:"Product category"`
	InStock     bool      `json:"in_stock" example:"true" description:"Stock availability"`
	CreatedAt   time.Time `json:"created_at" example:"2023-10-20T10:30:00Z" description:"Creation timestamp"`
	UpdatedAt   time.Time `json:"updated_at" example:"2023-10-20T10:30:00Z" description:"Last update timestamp"`
}

// SearchRequest represents a search request
type SearchRequest struct {
	Query  string `json:"query" validate:"required,min=1" example:"iPhone" description:"Search query"`
	Limit  int    `json:"limit" validate:"min=1,max=100" example:"20" description:"Maximum number of results"`
	Offset int    `json:"offset" validate:"min=0" example:"0" description:"Number of results to skip"`
}

// SearchResponse represents a search response
type SearchResponse struct {
	Results []ProductResponse `json:"results" description:"Search results"`
	Total   int64             `json:"total" example:"50" description:"Total number of results"`
	Query   string            `json:"query" example:"iPhone" description:"Search query used"`
}
