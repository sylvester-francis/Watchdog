package services

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"

	"github.com/sylvester-francis/watchdog/core/domain"
	"github.com/sylvester-francis/watchdog/core/ports"
	"github.com/sylvester-francis/watchdog/internal/crypto"
)

// Auth service errors.
var (
	ErrEmailAlreadyExists = errors.New("email already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserNotFound       = errors.New("user not found")
	ErrInvalidAPIKey      = errors.New("invalid API key")
	ErrAgentNotFound      = errors.New("agent not found")
)

// AuthService implements both ports.UserAuthService and ports.AgentAuthService.
type AuthService struct {
	userRepo       ports.UserRepository
	agentRepo      ports.AgentRepository
	usageEventRepo ports.UsageEventRepository
	hasher         *crypto.PasswordHasher
	encryptor      *crypto.Encryptor
	logger         *slog.Logger
}

// NewAuthService creates a new AuthService.
func NewAuthService(
	userRepo ports.UserRepository,
	agentRepo ports.AgentRepository,
	usageEventRepo ports.UsageEventRepository,
	hasher *crypto.PasswordHasher,
	encryptor *crypto.Encryptor,
	logger *slog.Logger,
) *AuthService {
	if logger == nil {
		logger = slog.Default()
	}
	return &AuthService{
		userRepo:       userRepo,
		agentRepo:      agentRepo,
		usageEventRepo: usageEventRepo,
		hasher:         hasher,
		encryptor:      encryptor,
		logger:         logger,
	}
}

// Register creates a new user account.
func (s *AuthService) Register(ctx context.Context, email, password string) (*domain.User, error) {
	exists, err := s.userRepo.ExistsByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("authService.Register: check email: %w", err)
	}
	if exists {
		return nil, ErrEmailAlreadyExists
	}

	passwordHash, err := s.hasher.Hash(password)
	if err != nil {
		return nil, fmt.Errorf("authService.Register: hash password: %w", err)
	}

	user := domain.NewUser(email, passwordHash)
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("authService.Register: create user: %w", err)
	}

	return user, nil
}

// Login authenticates a user by email and password.
func (s *AuthService) Login(ctx context.Context, email, password string) (*domain.User, error) {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("authService.Login: get user: %w", err)
	}
	if user == nil {
		return nil, ErrInvalidCredentials
	}

	valid, err := s.hasher.Verify(password, user.PasswordHash)
	if err != nil {
		return nil, fmt.Errorf("authService.Login: verify password: %w", err)
	}
	if !valid {
		return nil, ErrInvalidCredentials
	}

	return user, nil
}

// ValidateAPIKey validates an agent's API key and returns the agent if valid.
// API keys use the format "agentID:secret" for O(1) lookup by agent ID.
func (s *AuthService) ValidateAPIKey(ctx context.Context, apiKey string) (*domain.Agent, error) {
	parts := strings.SplitN(apiKey, ":", 2)
	if len(parts) != 2 {
		return nil, ErrInvalidAPIKey
	}

	agentID, err := uuid.Parse(parts[0])
	if err != nil {
		return nil, ErrInvalidAPIKey
	}
	secret := parts[1]

	agent, err := s.agentRepo.GetByID(ctx, agentID)
	if err != nil {
		return nil, fmt.Errorf("authService.ValidateAPIKey: get agent: %w", err)
	}
	if agent == nil {
		return nil, ErrInvalidAPIKey
	}

	decryptedKey, err := s.encryptor.DecryptString(agent.APIKeyEncrypted)
	if err != nil {
		return nil, fmt.Errorf("authService.ValidateAPIKey: decrypt key: %w", err)
	}

	if subtle.ConstantTimeCompare([]byte(secret), []byte(decryptedKey)) != 1 {
		return nil, ErrInvalidAPIKey
	}

	return agent, nil
}

// CreateAgent creates a new agent for a user and returns the full API key.
// The API key format is "agentID:secret" - shown once, must be saved by the user.
func (s *AuthService) CreateAgent(ctx context.Context, userID string, name string) (*domain.Agent, string, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, "", fmt.Errorf("authService.CreateAgent: invalid user ID: %w", err)
	}

	// Enforce plan limits
	user, err := s.userRepo.GetByID(ctx, uid)
	if err != nil {
		return nil, "", fmt.Errorf("authService.CreateAgent: get user: %w", err)
	}
	if user == nil {
		return nil, "", ErrUserNotFound
	}

	limits := user.Plan.Limits()
	if limits.MaxAgents != -1 {
		count, err := s.agentRepo.CountByUserID(ctx, uid)
		if err != nil {
			return nil, "", fmt.Errorf("authService.CreateAgent: count agents: %w", err)
		}
		if count >= limits.MaxAgents {
			event := domain.NewUsageEvent(uid, domain.EventLimitHit, domain.ResourceAgent, count, limits.MaxAgents, user.Plan)
			if err := s.usageEventRepo.Create(ctx, event); err != nil {
				s.logger.Warn("failed to record limit_hit event", "error", err)
			}
			return nil, "", domain.ErrAgentLimitReached
		}
		if float64(count) >= float64(limits.MaxAgents)*0.8 {
			event := domain.NewUsageEvent(uid, domain.EventApproachingLimit, domain.ResourceAgent, count, limits.MaxAgents, user.Plan)
			if err := s.usageEventRepo.Create(ctx, event); err != nil {
				s.logger.Warn("failed to record approaching_limit event", "error", err)
			}
		}
	}

	secret, err := domain.GenerateAPIKey()
	if err != nil {
		return nil, "", fmt.Errorf("authService.CreateAgent: generate API key: %w", err)
	}

	encryptedKey, err := s.encryptor.EncryptString(secret)
	if err != nil {
		return nil, "", fmt.Errorf("authService.CreateAgent: encrypt API key: %w", err)
	}

	agent := domain.NewAgent(uid, name, encryptedKey)
	if err := s.agentRepo.Create(ctx, agent); err != nil {
		return nil, "", fmt.Errorf("authService.CreateAgent: create agent: %w", err)
	}

	// Full API key = "agentID:secret" (only time it's available)
	fullKey := agent.ID.String() + ":" + secret

	return agent, fullKey, nil
}
