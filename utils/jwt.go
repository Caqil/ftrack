package utils

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JWTService struct {
	secretKey       []byte
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
}

type Claims struct {
	UserID    string `json:"userId"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	TokenType string `json:"tokenType"` // access, refresh
	jwt.RegisteredClaims
}

type TokenPair struct {
	AccessToken  string    `json:"accessToken"`
	RefreshToken string    `json:"refreshToken"`
	TokenType    string    `json:"tokenType"`
	ExpiresIn    int64     `json:"expiresIn"`
	ExpiresAt    time.Time `json:"expiresAt"`
}

func NewJWTService(secretKey string) *JWTService {
	return &JWTService{
		secretKey:       []byte(secretKey),
		accessTokenTTL:  15 * time.Minute,   // Short-lived access token
		refreshTokenTTL: 7 * 24 * time.Hour, // 7 days refresh token
	}
}

func (j *JWTService) GenerateTokenPair(userID, email, role string) (*TokenPair, error) {
	// Generate access token
	accessToken, err := j.generateToken(userID, email, role, "access", j.accessTokenTTL)
	if err != nil {
		return nil, err
	}

	// Generate refresh token
	refreshToken, err := j.generateToken(userID, email, role, "refresh", j.refreshTokenTTL)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(j.accessTokenTTL.Seconds()),
		ExpiresAt:    time.Now().Add(j.accessTokenTTL),
	}, nil
}

func (j *JWTService) generateToken(userID, email, role, tokenType string, ttl time.Duration) (string, error) {
	now := time.Now()
	expiresAt := now.Add(ttl)

	claims := Claims{
		UserID:    userID,
		Email:     email,
		Role:      role,
		TokenType: tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "ftrack",
			Subject:   userID,
			ID:        GenerateUUID(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(j.secretKey)
}

func (j *JWTService) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return j.secretKey, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

func (j *JWTService) RefreshToken(refreshTokenString string) (*TokenPair, error) {
	claims, err := j.ValidateToken(refreshTokenString)
	if err != nil {
		return nil, err
	}

	if claims.TokenType != "refresh" {
		return nil, errors.New("invalid token type")
	}

	// Generate new token pair
	return j.GenerateTokenPair(claims.UserID, claims.Email, claims.Role)
}

func (j *JWTService) RevokeToken(tokenString string) error {
	// In a production environment, you would store revoked tokens in Redis
	// For now, we'll just validate the token format
	_, err := j.ValidateToken(tokenString)
	return err
}

func (j *JWTService) ExtractUserID(tokenString string) (string, error) {
	claims, err := j.ValidateToken(tokenString)
	if err != nil {
		return "", err
	}
	return claims.UserID, nil
}
