package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/nicholemattera/serenity/internal/models"
	"github.com/nicholemattera/serenity/internal/repository"
)

type Claims struct {
	jwt.RegisteredClaims
	UserID uuid.UUID `json:"user_id"`
	RoleID uuid.UUID `json:"role_id"`
}

type AuthService interface {
	Login(ctx context.Context, email, password string) (string, error)
	ValidateToken(token string) (*Claims, error)
}

type authService struct {
	userRepo   repository.UserRepository
	roleRepo   repository.RoleRepository
	secret     []byte
	issuer     string
	audience   string
	bcryptCost int
}

func NewAuthService(userRepo repository.UserRepository, roleRepo repository.RoleRepository, jwtSecret, issuer, audience string, bcryptCost int) AuthService {
	return &authService{
		userRepo:   userRepo,
		roleRepo:   roleRepo,
		secret:     []byte(jwtSecret),
		issuer:     issuer,
		audience:   audience,
		bcryptCost: bcryptCost,
	}
}

func (s *authService) Login(ctx context.Context, email, password string) (string, error) {
	dummyHash, err := bcrypt.GenerateFromPassword([]byte("dummy_password"), s.bcryptCost)
	if err != nil {
		return "", errors.New("internal error")
	}

	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		// Prevents timing to know whether or not a user exists.
		err := bcrypt.CompareHashAndPassword([]byte(dummyHash), []byte(password))
		if err != nil {
			return "", ErrUnauthorized
		}

		return "", ErrUnauthorized
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", ErrUnauthorized
	}

	role, err := s.roleRepo.GetByID(ctx, user.RoleID)
	if err != nil {
		return "", fmt.Errorf("failed to load role: %w", err)
	}

	return s.issueToken(user, role)
}

func (s *authService) issueToken(user *models.User, role *models.Role) (string, error) {
	expiry := time.Now().Add(time.Duration(role.SessionTimeout) * time.Second)

	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID.String(),
			Issuer:    s.issuer,
			Audience:  jwt.ClaimStrings{s.audience},
			ExpiresAt: jwt.NewNumericDate(expiry),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		UserID: user.ID,
		RoleID: user.RoleID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.secret)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return signed, nil
}

func (s *authService) ValidateToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.secret, nil
	}, jwt.WithIssuer(s.issuer), jwt.WithAudience(s.audience))
	if err != nil {
		return nil, ErrUnauthorized
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrUnauthorized
	}

	return claims, nil
}
