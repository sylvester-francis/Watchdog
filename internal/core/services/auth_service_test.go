package services_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sylvester-francis/watchdog/core/domain"
	"github.com/sylvester-francis/watchdog/internal/core/services"
	"github.com/sylvester-francis/watchdog/internal/crypto"
	"github.com/sylvester-francis/watchdog/internal/testutil/mocks"
)

const testEncryptionKey = "01234567890123456789012345678901" // 32 bytes

func newTestAuthService(userRepo *mocks.MockUserRepository, agentRepo *mocks.MockAgentRepository) *services.AuthService {
	hasher := crypto.NewPasswordHasher()
	encryptor, _ := crypto.NewEncryptor(testEncryptionKey)
	return services.NewAuthService(userRepo, agentRepo, &mocks.MockUsageEventRepository{}, hasher, encryptor, nil)
}

// --- Register ---

func TestRegister_Success(t *testing.T) {
	var createdUser *domain.User
	userRepo := &mocks.MockUserRepository{
		ExistsByEmailFn: func(_ context.Context, _ string) (bool, error) {
			return false, nil
		},
		CreateFn: func(_ context.Context, user *domain.User) error {
			createdUser = user
			return nil
		},
	}
	svc := newTestAuthService(userRepo, &mocks.MockAgentRepository{})

	user, err := svc.Register(context.Background(), "test@example.com", "password123")

	require.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, "test@example.com", user.Email)
	assert.NotEmpty(t, user.PasswordHash)
	assert.NotEqual(t, "password123", user.PasswordHash)
	assert.Equal(t, createdUser, user)
}

func TestRegister_EmailExists(t *testing.T) {
	userRepo := &mocks.MockUserRepository{
		ExistsByEmailFn: func(_ context.Context, _ string) (bool, error) {
			return true, nil
		},
	}
	svc := newTestAuthService(userRepo, &mocks.MockAgentRepository{})

	user, err := svc.Register(context.Background(), "test@example.com", "password123")

	assert.Nil(t, user)
	assert.ErrorIs(t, err, services.ErrEmailAlreadyExists)
}

func TestRegister_RepoError(t *testing.T) {
	repoErr := errors.New("db connection failed")
	userRepo := &mocks.MockUserRepository{
		ExistsByEmailFn: func(_ context.Context, _ string) (bool, error) {
			return false, nil
		},
		CreateFn: func(_ context.Context, _ *domain.User) error {
			return repoErr
		},
	}
	svc := newTestAuthService(userRepo, &mocks.MockAgentRepository{})

	user, err := svc.Register(context.Background(), "test@example.com", "password123")

	assert.Nil(t, user)
	assert.ErrorIs(t, err, repoErr)
}

// --- Login ---

func TestLogin_Success(t *testing.T) {
	hasher := crypto.NewPasswordHasher()
	hash, _ := hasher.Hash("correctpassword")

	userRepo := &mocks.MockUserRepository{
		GetByEmailFn: func(_ context.Context, _ string) (*domain.User, error) {
			return &domain.User{
				ID:           uuid.New(),
				Email:        "test@example.com",
				PasswordHash: hash,
			}, nil
		},
	}
	svc := newTestAuthService(userRepo, &mocks.MockAgentRepository{})

	user, err := svc.Login(context.Background(), "test@example.com", "correctpassword")

	require.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, "test@example.com", user.Email)
}

func TestLogin_UserNotFound(t *testing.T) {
	userRepo := &mocks.MockUserRepository{
		GetByEmailFn: func(_ context.Context, _ string) (*domain.User, error) {
			return nil, nil
		},
	}
	svc := newTestAuthService(userRepo, &mocks.MockAgentRepository{})

	user, err := svc.Login(context.Background(), "noone@example.com", "password123")

	assert.Nil(t, user)
	assert.ErrorIs(t, err, services.ErrInvalidCredentials)
}

func TestLogin_WrongPassword(t *testing.T) {
	hasher := crypto.NewPasswordHasher()
	hash, _ := hasher.Hash("correctpassword")

	userRepo := &mocks.MockUserRepository{
		GetByEmailFn: func(_ context.Context, _ string) (*domain.User, error) {
			return &domain.User{
				ID:           uuid.New(),
				Email:        "test@example.com",
				PasswordHash: hash,
			}, nil
		},
	}
	svc := newTestAuthService(userRepo, &mocks.MockAgentRepository{})

	user, err := svc.Login(context.Background(), "test@example.com", "wrongpassword")

	assert.Nil(t, user)
	assert.ErrorIs(t, err, services.ErrInvalidCredentials)
}

// --- ValidateAPIKey ---

func TestValidateAPIKey_Success(t *testing.T) {
	encryptor, _ := crypto.NewEncryptor(testEncryptionKey)
	secret := "deadbeef1234567890abcdef12345678"
	encrypted, _ := encryptor.EncryptString(secret)
	agentID := uuid.New()

	agentRepo := &mocks.MockAgentRepository{
		GetByIDFn: func(_ context.Context, id uuid.UUID) (*domain.Agent, error) {
			assert.Equal(t, agentID, id)
			return &domain.Agent{
				ID:              agentID,
				APIKeyEncrypted: encrypted,
			}, nil
		},
	}
	svc := newTestAuthService(&mocks.MockUserRepository{}, agentRepo)

	agent, err := svc.ValidateAPIKey(context.Background(), agentID.String()+":"+secret)

	require.NoError(t, err)
	assert.NotNil(t, agent)
	assert.Equal(t, agentID, agent.ID)
}

func TestValidateAPIKey_InvalidFormat(t *testing.T) {
	svc := newTestAuthService(&mocks.MockUserRepository{}, &mocks.MockAgentRepository{})

	agent, err := svc.ValidateAPIKey(context.Background(), "no-colon-separator")

	assert.Nil(t, agent)
	assert.ErrorIs(t, err, services.ErrInvalidAPIKey)
}

func TestValidateAPIKey_InvalidUUID(t *testing.T) {
	svc := newTestAuthService(&mocks.MockUserRepository{}, &mocks.MockAgentRepository{})

	agent, err := svc.ValidateAPIKey(context.Background(), "not-a-uuid:somesecret")

	assert.Nil(t, agent)
	assert.ErrorIs(t, err, services.ErrInvalidAPIKey)
}

func TestValidateAPIKey_AgentNotFound(t *testing.T) {
	agentRepo := &mocks.MockAgentRepository{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Agent, error) {
			return nil, nil
		},
	}
	svc := newTestAuthService(&mocks.MockUserRepository{}, agentRepo)

	agent, err := svc.ValidateAPIKey(context.Background(), uuid.New().String()+":somesecret")

	assert.Nil(t, agent)
	assert.ErrorIs(t, err, services.ErrInvalidAPIKey)
}

func TestValidateAPIKey_WrongSecret(t *testing.T) {
	encryptor, _ := crypto.NewEncryptor(testEncryptionKey)
	encrypted, _ := encryptor.EncryptString("correctsecret")
	agentID := uuid.New()

	agentRepo := &mocks.MockAgentRepository{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Agent, error) {
			return &domain.Agent{
				ID:              agentID,
				APIKeyEncrypted: encrypted,
			}, nil
		},
	}
	svc := newTestAuthService(&mocks.MockUserRepository{}, agentRepo)

	agent, err := svc.ValidateAPIKey(context.Background(), agentID.String()+":wrongsecret")

	assert.Nil(t, agent)
	assert.ErrorIs(t, err, services.ErrInvalidAPIKey)
}

// --- CreateAgent ---

func TestCreateAgent_Success(t *testing.T) {
	userID := uuid.New()
	userRepo := &mocks.MockUserRepository{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.User, error) {
			return &domain.User{ID: userID, Plan: domain.PlanBeta}, nil
		},
	}
	agentRepo := &mocks.MockAgentRepository{
		CountByUserIDFn: func(_ context.Context, _ uuid.UUID) (int, error) {
			return 0, nil
		},
		CreateFn: func(_ context.Context, _ *domain.Agent) error {
			return nil
		},
	}
	svc := newTestAuthService(userRepo, agentRepo)

	agent, apiKey, err := svc.CreateAgent(context.Background(), userID.String(), "test-agent")

	require.NoError(t, err)
	assert.NotNil(t, agent)
	assert.Equal(t, "test-agent", agent.Name)
	assert.Equal(t, userID, agent.UserID)
	assert.NotEmpty(t, apiKey)

	// API key format: "agentID:secret"
	parts := strings.SplitN(apiKey, ":", 2)
	assert.Len(t, parts, 2)
	assert.Equal(t, agent.ID.String(), parts[0])
	assert.NotEmpty(t, parts[1])
}

func TestCreateAgent_InvalidUserID(t *testing.T) {
	svc := newTestAuthService(&mocks.MockUserRepository{}, &mocks.MockAgentRepository{})

	agent, apiKey, err := svc.CreateAgent(context.Background(), "not-a-uuid", "test-agent")

	assert.Nil(t, agent)
	assert.Empty(t, apiKey)
	assert.Error(t, err)
}

func TestCreateAgent_RepoError(t *testing.T) {
	userID := uuid.New()
	repoErr := errors.New("db error")
	userRepo := &mocks.MockUserRepository{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.User, error) {
			return &domain.User{ID: userID, Plan: domain.PlanBeta}, nil
		},
	}
	agentRepo := &mocks.MockAgentRepository{
		CountByUserIDFn: func(_ context.Context, _ uuid.UUID) (int, error) {
			return 0, nil
		},
		CreateFn: func(_ context.Context, _ *domain.Agent) error {
			return repoErr
		},
	}
	svc := newTestAuthService(userRepo, agentRepo)

	agent, apiKey, err := svc.CreateAgent(context.Background(), userID.String(), "test-agent")

	assert.Nil(t, agent)
	assert.Empty(t, apiKey)
	assert.ErrorIs(t, err, repoErr)
}
