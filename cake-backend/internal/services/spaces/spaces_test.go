package spaces

import (
	"errors"
	"fmt"
	"testing"
	"time"
	"uuid"

	"efournierrobert/cake-backend/internal/handlers/handler_errors"
	"efournierrobert/cake-backend/internal/handlers/models"
	"efournierrobert/cake-backend/internal/repository/repo_errors"
	spaceRepo "efournierrobert/cake-backend/internal/repository/spaces"
	userRepo "efournierrobert/cake-backend/internal/repository/users"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeSpaceRepository struct {
	allSpaces []spaceRepo.Space
	allErr    error
	allCalls  int

	space      spaceRepo.Space
	getErr     error
	getOwner   uuid.UUID
	getSpaceID uuid.UUID
	getCalls   int

	createdSpace spaceRepo.Space
	createErr    error

	updatedSpace spaceRepo.Space
	updateErr    error

	deleteOwner   uuid.UUID
	deleteSpaceID uuid.UUID
	deleteErr     error
}

func (f *fakeSpaceRepository) GetAllUserSpaces(ownerUuid uuid.UUID) ([]spaceRepo.Space, error) {
	f.allCalls++
	return f.allSpaces, f.allErr
}

func (f *fakeSpaceRepository) GetUserSpace(ownerUuid, spaceUuid uuid.UUID) (spaceRepo.Space, error) {
	f.getCalls++
	f.getOwner = ownerUuid
	f.getSpaceID = spaceUuid
	return f.space, f.getErr
}

func (f *fakeSpaceRepository) CreateSpace(space spaceRepo.Space) (spaceRepo.Space, error) {
	f.createdSpace = space
	return space, f.createErr
}

func (f *fakeSpaceRepository) UpdateSpace(space spaceRepo.Space) (spaceRepo.Space, error) {
	f.updatedSpace = space
	return space, f.updateErr
}

func (f *fakeSpaceRepository) DeleteSpace(ownerUuid, spaceUuid uuid.UUID) error {
	f.deleteOwner = ownerUuid
	f.deleteSpaceID = spaceUuid
	return f.deleteErr
}

type fakeUserRepository struct {
	user  userRepo.User
	err   error
	calls int
	uuid  uuid.UUID
}

func (f *fakeUserRepository) GetUser(userUuid uuid.UUID) (userRepo.User, error) {
	f.calls++
	f.uuid = userUuid
	return f.user, f.err
}

func (f *fakeUserRepository) GetAllUsers() ([]userRepo.UserWithRole, error) {
	panic("unexpected call to GetAllUsers")
}

func (f *fakeUserRepository) UpdateUser(user userRepo.User) (userRepo.User, error) {
	panic("unexpected call to UpdateUser")
}

func (f *fakeUserRepository) CreateUser(user userRepo.User) (userRepo.User, error) {
	panic("unexpected call to CreateUser")
}

func (f *fakeUserRepository) DeleteUser(userUuid uuid.UUID) error {
	panic("unexpected call to DeleteUser")
}

func (f *fakeUserRepository) GetUserCredentials(username string) (userRepo.UserWithRole, error) {
	panic("unexpected call to GetUserCredentials")
}

func newTestService() (*Service, *fakeSpaceRepository, *fakeUserRepository) {
	spaces := &fakeSpaceRepository{}
	users := &fakeUserRepository{}
	return &Service{repo: spaces, userRepo: users}, spaces, users
}

func sampleSpace() spaceRepo.Space {
	return spaceRepo.Space{
		ID:           37,
		Uuid:         uuid.NewV4().String(),
		Name:         "Research",
		SystemPrompt: "Be concise",
		CreatedAt:    time.Date(2026, time.October, 1, 12, 30, 0, 0, time.UTC),
		UpdatedAt:    time.Date(2026, time.October, 2, 12, 30, 0, 0, time.UTC),
		OwnerId:      9,
	}
}

func TestGetAllSpaces(t *testing.T) {
	t.Run("maps spaces and returns an empty array when there are no spaces", func(t *testing.T) {
		svc, repo, _ := newTestService()
		ownerID := uuid.NewV4()
		row := sampleSpace()
		repo.allSpaces = []spaceRepo.Space{row}

		got, err := svc.GetAllSpaces(ownerID.String())
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, row.Uuid, got[0].Uuid)
		assert.Equal(t, row.Name, got[0].Name)
		assert.Equal(t, row.SystemPrompt, got[0].SystemPrompt)
		assert.Equal(t, row.CreatedAt, got[0].CreatedAt)
		assert.Equal(t, row.UpdatedAt, got[0].UpdatedAt)

		repo.allSpaces = nil
		got, err = svc.GetAllSpaces(ownerID.String())
		require.NoError(t, err)
		assert.NotNil(t, got, "empty results should encode as an empty array, not null")
		assert.Empty(t, got)
		assert.Equal(t, 2, repo.allCalls)
	})

	t.Run("rejects an invalid owner UUID", func(t *testing.T) {
		svc, repo, _ := newTestService()
		_, err := svc.GetAllSpaces("not-a-uuid")
		assert.ErrorIs(t, err, handler_errors.ErrInvalidRequest)
		assert.Zero(t, repo.allCalls)
	})

	t.Run("maps repository failures to an unexpected error", func(t *testing.T) {
		svc, repo, _ := newTestService()
		repo.allErr = errors.New("database unavailable")
		_, err := svc.GetAllSpaces(uuid.NewV4().String())
		assert.ErrorIs(t, err, handler_errors.ErrUnexpectedError)
	})
}

func TestGetSpace(t *testing.T) {
	svc, repo, _ := newTestService()
	ownerID := uuid.NewV4()
	row := sampleSpace()
	repo.space = row

	got, err := svc.GetSpace(row.Uuid, ownerID.String())
	require.NoError(t, err)
	assert.Equal(t, row.Uuid, got.Uuid)
	assert.Equal(t, row.Name, got.Name)
	assert.Equal(t, ownerID, repo.getOwner)
	assert.Equal(t, uuid.MustParse(row.Uuid), repo.getSpaceID)

	_, err = svc.GetSpace("bad-space-id", ownerID.String())
	assert.ErrorIs(t, err, handler_errors.ErrInvalidRequest)
	_, err = svc.GetSpace(row.Uuid, "bad-owner-id")
	assert.ErrorIs(t, err, handler_errors.ErrInvalidRequest)
	assert.Equal(t, 1, repo.getCalls, "invalid UUIDs should not reach the repository")

	repo.getErr = &repo_errors.SpaceNotFound{}
	_, err = svc.GetSpace(row.Uuid, ownerID.String())
	assert.ErrorIs(t, err, handler_errors.ErrSpaceNotFound)

	repo.getErr = errors.New("database unavailable")
	_, err = svc.GetSpace(row.Uuid, ownerID.String())
	assert.ErrorIs(t, err, handler_errors.ErrUnexpectedError)
}

func TestCreateSpace(t *testing.T) {
	svc, repo, users := newTestService()
	ownerID := uuid.NewV4()
	users.user = userRepo.User{Id: 42, Uuid: ownerID.String()}

	got, err := svc.CreateSpace(ownerID.String(), models.SpaceCreate{
		Name:         "My space",
		SystemPrompt: "Answer in short sentences",
	})
	require.NoError(t, err)
	assert.Equal(t, "My space", repo.createdSpace.Name)
	assert.Equal(t, "Answer in short sentences", repo.createdSpace.SystemPrompt)
	assert.Equal(t, 42, repo.createdSpace.OwnerId)
	assert.Equal(t, ownerID, users.uuid)
	_, parseErr := uuid.Parse(repo.createdSpace.Uuid)
	assert.NoError(t, parseErr)
	assert.Equal(t, repo.createdSpace.Uuid, got.Uuid)
	assert.Equal(t, repo.createdSpace.Name, got.Name)

	_, err = svc.CreateSpace(ownerID.String(), models.SpaceCreate{Name: "No prompt"})
	require.NoError(t, err)
	assert.Empty(t, repo.createdSpace.SystemPrompt, "an omitted create prompt should remain an empty string")

	_, err = svc.CreateSpace("bad-owner-id", models.SpaceCreate{Name: "Invalid"})
	assert.ErrorIs(t, err, handler_errors.ErrInvalidRequest)
	assert.Equal(t, 2, users.calls, "invalid UUIDs should not reach the user repository")

	users.err = &repo_errors.UserNotFound{}
	_, err = svc.CreateSpace(ownerID.String(), models.SpaceCreate{Name: "Missing owner"})
	assert.ErrorIs(t, err, handler_errors.ErrUserDoesNotExist)

	users.err = nil
	repo.createErr = errors.New("database unavailable")
	_, err = svc.CreateSpace(ownerID.String(), models.SpaceCreate{Name: "Database error"})
	assert.ErrorIs(t, err, handler_errors.ErrUnexpectedError)
}

func TestUpdateSpace(t *testing.T) {
	svc, repo, _ := newTestService()
	ownerID := uuid.NewV4()
	row := sampleSpace()
	repo.space = row
	newPrompt := ""

	got, err := svc.UpdateSpace(ownerID.String(), row.Uuid, models.SpaceUpdate{
		SystemPrompt: &newPrompt,
	})
	require.NoError(t, err)
	assert.Equal(t, row.Name, repo.updatedSpace.Name, "nil fields should be left unchanged")
	assert.Empty(t, repo.updatedSpace.SystemPrompt, "an explicit empty string should be stored")
	assert.Equal(t, row.Uuid, repo.updatedSpace.Uuid)
	assert.Equal(t, row.OwnerId, repo.updatedSpace.OwnerId)
	assert.Empty(t, got.SystemPrompt)

	updatedName := "Updated research"
	repo.space = row
	_, err = svc.UpdateSpace(ownerID.String(), row.Uuid, models.SpaceUpdate{Name: &updatedName})
	require.NoError(t, err)
	assert.Equal(t, updatedName, repo.updatedSpace.Name)
	assert.Equal(t, row.SystemPrompt, repo.updatedSpace.SystemPrompt)

	_, err = svc.UpdateSpace("bad-owner-id", row.Uuid, models.SpaceUpdate{})
	assert.ErrorIs(t, err, handler_errors.ErrInvalidRequest)
	_, err = svc.UpdateSpace(ownerID.String(), "bad-space-id", models.SpaceUpdate{})
	assert.ErrorIs(t, err, handler_errors.ErrInvalidRequest)

	repo.getErr = &repo_errors.SpaceNotFound{}
	_, err = svc.UpdateSpace(ownerID.String(), row.Uuid, models.SpaceUpdate{})
	assert.ErrorIs(t, err, handler_errors.ErrSpaceNotFound)

	repo.getErr = nil
	repo.space = row
	repo.updateErr = errors.New("database unavailable")
	_, err = svc.UpdateSpace(ownerID.String(), row.Uuid, models.SpaceUpdate{Name: &updatedName})
	assert.ErrorIs(t, err, handler_errors.ErrUnexpectedError)
}

func TestDeleteSpace(t *testing.T) {
	svc, repo, _ := newTestService()
	ownerID := uuid.NewV4()
	spaceID := uuid.NewV4()

	require.NoError(t, svc.DeleteSpace(ownerID.String(), spaceID.String()))
	assert.Equal(t, ownerID, repo.deleteOwner)
	assert.Equal(t, spaceID, repo.deleteSpaceID)

	assert.ErrorIs(t, svc.DeleteSpace("bad-owner-id", spaceID.String()), handler_errors.ErrInvalidRequest)
	assert.ErrorIs(t, svc.DeleteSpace(ownerID.String(), "bad-space-id"), handler_errors.ErrInvalidRequest)

	repo.deleteErr = &repo_errors.SpaceNotFound{}
	assert.ErrorIs(t, svc.DeleteSpace(ownerID.String(), spaceID.String()), handler_errors.ErrSpaceNotFound)

	repo.deleteErr = errors.New("database unavailable")
	assert.ErrorIs(t, svc.DeleteSpace(ownerID.String(), spaceID.String()), handler_errors.ErrUnexpectedError)
}

func TestGetAppErrorTypeUnwrapsRepositoryErrors(t *testing.T) {
	wrapped := fmt.Errorf("repository call: %w", &repo_errors.SpaceNotFound{})
	assert.ErrorIs(t, getAppErrorType(wrapped), handler_errors.ErrSpaceNotFound)
	assert.ErrorIs(t,
		getAppErrorType(fmt.Errorf("repository call: %w", &repo_errors.UserNotFound{})),
		handler_errors.ErrUserDoesNotExist,
	)
	assert.ErrorIs(t, getAppErrorType(errors.New("other failure")), handler_errors.ErrUnexpectedError)
}
