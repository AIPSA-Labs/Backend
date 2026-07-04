package jwt

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrTokenExpired = errors.New("token expired")
)

type Claims struct {
	UserID    uuid.UUID `json:"user_id"`
	Email     string    `json:"email"`
	TokenType string    `json:"token_type"`
	jwt.RegisteredClaims
}

type JWTManager struct {
	secret          []byte
	accessDuration  time.Duration
	refreshDuration time.Duration
}

func NewManager(secret string, accessDuration, refreshDuration time.Duration) *JWTManager {
	return &JWTManager{
		secret:          []byte(secret),
		accessDuration:  accessDuration,
		refreshDuration: refreshDuration,
	}
}

func (m *JWTManager) GenerateAccessToken(userID uuid.UUID, email string) (string, error) {
	return m.generateToken(userID, email, "access", m.accessDuration)
}

func (m *JWTManager) GenerateRefreshToken(userID uuid.UUID, email string) (string, error) {
	return m.generateToken(userID, email, "refresh", m.refreshDuration)
}

func (m *JWTManager) generateToken(userID uuid.UUID, email, tokenType string, duration time.Duration) (string, error) {
	claims := Claims{
		UserID:    userID,
		Email:     email,
		TokenType: tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(duration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   userID.String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

func (m *JWTManager) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return m.secret, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}
