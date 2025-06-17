// types/token_pair.go
package types
import "github.com/princeprakhar/ecommerce-backend/internal/models"

type TokenPair struct {
	AccessToken           string `json:"access_token"`
	RefreshToken          string `json:"refresh_token"`
	AccessTokenExpiresAt  int64  `json:"access_token_expires_at"`
	RefreshTokenExpiresAt int64  `json:"refresh_token_expires_at"`
}

type AuthResponse struct {
	Token TokenPair     `json:"token"`
	User  models.User   `json:"user"`
}
