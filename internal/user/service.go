package user

import (
	"github.com/golang-jwt/jwt/v5"
)

// Service contains the business logic for the /me endpoint.
type Service struct{}

// NewService creates a new me Service.
func NewService() *Service {
	return &Service{}
}

// GetProfile returns the user's profile data from JWT claims.
func (s *Service) GetProfile(claims jwt.MapClaims) map[string]interface{} {
	return claims
}
