package users

import (
	"context"
	"log"
	"os"
	"testing"
	"time"
	"uuid"

	"efournierrobert/cake-backend/internal/repository"
	"efournierrobert/cake-backend/internal/repository/repo_errors"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/mariadb"
)

// testAdminRoleUUID is the uuid of the seeded admin user role from
// the initial migration.
const testAdminRoleUUID = "5a18559d-9d20-4251-8a72-b36efbaff514"

var testDB *sqlx.DB

// TestMain starts a fresh MariaDB container, points DATABASE_URL at
// it and opens the connection through repository.NewDbConnection so
// all pending migrations run, exactly like the application does.
// testDB holds that migrated connection for the whole test binary.
func TestMain(m *testing.M) {
	ctx := context.Background()

	container, err := mariadb.Run(ctx, "mariadb:latest",
		mariadb.WithDatabase("cake"),
		mariadb.WithUsername("cake-user"),
		mariadb.WithPassword("cake-user"),
	)
	if err != nil {
		log.Fatalf("test setup: failed to start the MariaDB testcontainer: %v", err)
	}
	defer container.Terminate(ctx)

	// The MariaDB server runs in UTC; set loc so the driver decodes
	// DATETIME columns into the same instant time.Now() measures in
	// the host's timezone.
	dsn, err := container.ConnectionString(ctx, "parseTime=true", "loc=UTC")
	if err != nil {
		log.Fatalf("test setup: failed to build the DATABASE_URL from the container endpoint: %v", err)
	}
	os.Setenv("DATABASE_URL", dsn)

	// NewDbConnection resolves "migrations" relative to the working
	// directory, which is this package's directory. Cd to the
	// repository root so the migrations directory is found, no matter
	// where the test binary is launched from.
	if _, statErr := os.Stat("migrations"); os.IsNotExist(statErr) {
		// This package sits three levels below the module root.
		if err := os.Chdir("../../.."); err != nil {
			log.Fatalf("test setup: failed to chdir into the module root: %v", err)
		}
	}

	db, err := repository.NewDbConnection()
	if err != nil {
		if db != nil {
			db.Close()
		}
		log.Fatalf("test setup: %v", err)
	}
	defer db.Close()

	testDB = db

	code := m.Run()
	os.Exit(code)
}

// seededAdminRoleID looks up the id of the seeded admin user role.
func seededAdminRoleID(t *testing.T) int {
	t.Helper()
	var roleID int
	err := testDB.QueryRow(
		"SELECT id FROM user_roles WHERE uuid = ?", testAdminRoleUUID,
	).Scan(&roleID)
	require.NoError(t, err, "expected the seeded admin role to exist")
	return roleID
}

// makeUser builds a user with a fresh random uuid and the seeded
// user role, ready to be created or used as a foreign key reference.
func makeUser(t *testing.T, username, firstName, lastName string) User {
	t.Helper()
	return User{
		Uuid:         uuid.NewV4().String(),
		Username:     username,
		PasswordHash: []byte("$2b$hash$" + username),
		FirstName:    firstName,
		LastName:     lastName,
		RoleId:       2,
	}
}

// insertOwnedConversation inserts a space owned by the user and a
// conversation in it, directly through the test database, since no
// conversation or space repository exists yet. It returns the
// conversation id.
func insertOwnedConversation(t *testing.T, owner User) int {
	t.Helper()

	var spaceID, conversationID int

	_, err := testDB.Exec(
		"INSERT INTO spaces (uuid, name, owner_id, created_at, updated_at) VALUES (?, ?, ?, NOW(), NOW())",
		uuid.NewV4().String(), "default space", owner.Id,
	)
	require.NoError(t, err, "failed to insert a space owned by the user")
	err = testDB.QueryRow(
		"SELECT id FROM spaces WHERE owner_id = ? AND name = ?", owner.Id, "default space",
	).Scan(&spaceID)
	require.NoError(t, err, "failed to look up the inserted space")

	_, err = testDB.Exec(
		"INSERT INTO conversations (uuid, owner_id, space_id, title, created_at) VALUES (?, ?, ?, ?, NOW())",
		uuid.NewV4().String(), owner.Id, spaceID, "owned conversation",
	)
	require.NoError(t, err, "failed to insert a conversation in the owned space")
	err = testDB.QueryRow(
		"SELECT id FROM conversations WHERE owner_id = ? AND space_id = ?", owner.Id, spaceID,
	).Scan(&conversationID)
	require.NoError(t, err, "failed to look up the inserted conversation")

	return conversationID
}

// countRows runs the given query and returns the single-column count
// it produces.
func countRows(t *testing.T, query string, args ...any) int {
	t.Helper()
	var count int
	err := testDB.QueryRow(query, args...).Scan(&count)
	require.NoError(t, err, "failed to run %q", query)
	return count
}

func TestCreateUser(t *testing.T) {
	repo := New(testDB)
	user := makeUser(t, "create-user", "Ada", "Lovelace")

	created, err := repo.CreateUser(user)
	require.NoError(t, err, "expected CreateUser to succeed")

	assert.Positive(t, created.Id, "expected a positive auto-generated id")
	assert.Equal(t, user.Uuid, created.Uuid)
	assert.Equal(t, user.Username, created.Username)
	assert.Equal(t, user.PasswordHash, created.PasswordHash)
	assert.Equal(t, user.FirstName, created.FirstName)
	assert.Equal(t, user.LastName, created.LastName)
	assert.Equal(t, user.RoleId, created.RoleId)
	assert.False(t, created.CreatedAt.IsZero(), "expected created_at to be populated")
	assert.False(t, created.UpdatedAt.IsZero(), "expected updated_at to be populated")
}

func TestCreateUserDuplicateUsername(t *testing.T) {
	repo := New(testDB)
	first := makeUser(t, "twin-user", "Ada", "Lovelace")
	second := makeUser(t, "twin-user", "Grace", "Hopper")
	second.RoleId = seededAdminRoleID(t)

	_, err := repo.CreateUser(first)
	require.NoError(t, err, "expected the first twin user to be created")

	_, err = repo.CreateUser(second)
	assert.ErrorIs(t, err, &repo_errors.UserAlreadyExists{}, "expected *repo_errors.UserAlreadyExists for a duplicate username")
}

func TestGetUserNotFound(t *testing.T) {
	repo := New(testDB)
	missing := uuid.NewV4()

	_, err := repo.GetUser(missing)
	assert.ErrorIs(t, err, &repo_errors.UserNotFound{}, "expected *repo_errors.UserNotFound for a missing uuid")
}

func TestUpdateUser(t *testing.T) {
	repo := New(testDB)
	user := makeUser(t, "update-user", "Ada", "Lovelace")
	_, err := repo.CreateUser(user)
	require.NoError(t, err, "expected CreateUser to succeed")

	// MariaDB rounds DATETIME values to seconds, so updated_at can only
	// be guaranteed to move forward after at least one second.
	time.Sleep(1100 * time.Millisecond)

	user.FirstName = "Augusta"
	user.LastName = "King"
	user.Username = "update-user-renamed"
	user.RoleId = seededAdminRoleID(t)

	updated, err := repo.UpdateUser(user)
	require.NoError(t, err, "expected UpdateUser to succeed")

	assert.Positive(t, updated.Id, "expected a positive auto-generated id")
	assert.Equal(t, "Augusta", updated.FirstName, "expected updated fields to be persisted")
	assert.Equal(t, "King", updated.LastName, "expected updated fields to be persisted")
	assert.Equal(t, "update-user-renamed", updated.Username, "expected updated fields to be persisted")
	assert.Equal(t, user.RoleId, updated.RoleId, "expected role id %d to be persisted", user.RoleId)
	assert.False(t, updated.UpdatedAt.Before(updated.CreatedAt),
		"expected updated_at (%v) to be after created_at (%v)", updated.UpdatedAt, updated.CreatedAt)
	assert.NotEqual(t, updated.CreatedAt.Unix(), updated.UpdatedAt.Unix(),
		"expected updated_at (%v) to differ from created_at (%v)", updated.UpdatedAt, updated.CreatedAt)

	_, err = repo.UpdateUser(makeUser(t, "ghost-user", "Nope", "Nope"))
	assert.ErrorIs(t, err, &repo_errors.UserNotFound{}, "expected *repo_errors.UserNotFound when updating a missing user")
}

func TestDeleteUser(t *testing.T) {
	repo := New(testDB)
	user := makeUser(t, "deleteme", "Ada", "Lovelace")
	_, err := repo.CreateUser(user)
	require.NoError(t, err, "expected CreateUser to succeed")

	require.NoError(t, repo.DeleteUser(uuid.MustParse(user.Uuid)), "expected DeleteUser to succeed")

	_, err = repo.GetUser(uuid.MustParse(user.Uuid))
	assert.ErrorIs(t, err, &repo_errors.UserNotFound{}, "expected *repo_errors.UserNotFound after deletion")

	err = repo.DeleteUser(uuid.MustParse(user.Uuid))
	assert.ErrorIs(t, err, &repo_errors.UserNotFound{}, "expected *repo_errors.UserNotFound when deleting an already deleted user")
}

func TestDeleteUserCascadesConversations(t *testing.T) {
	repo := New(testDB)
	user := makeUser(t, "cascade-user", "Ada", "Lovelace")
	created, err := repo.CreateUser(user)
	require.NoError(t, err, "expected CreateUser to succeed")
	conversationID := insertOwnedConversation(t, created)

	assert.Equal(t, 1, countRows(t, "SELECT COUNT(*) FROM spaces WHERE owner_id = ?", created.Id),
		"expected 1 owned space before deletion")
	assert.Equal(t, 1, countRows(t, "SELECT COUNT(*) FROM conversations WHERE owner_id = ?", created.Id),
		"expected 1 owned conversation before deletion")

	require.NoError(t, repo.DeleteUser(uuid.MustParse(created.Uuid)), "expected DeleteUser to succeed")

	assert.Equal(t, 0, countRows(t, "SELECT COUNT(*) FROM spaces WHERE owner_id = ?", created.Id),
		"expected the owned spaces to be cascade deleted")
	assert.Equal(t, 0, countRows(t, "SELECT COUNT(*) FROM conversations WHERE id = ?", conversationID),
		"expected the conversation to be cascade deleted through its owner")
}

func TestGetAllUsersOrdering(t *testing.T) {
	repo := New(testDB)
	first := makeUser(t, "oldest-user", "Ada", "Lovelace")
	_, err := repo.CreateUser(first)
	require.NoError(t, err, "expected CreateUser to succeed")

	time.Sleep(1100 * time.Millisecond)

	second := makeUser(t, "newest-user", "Grace", "Hopper")
	_, err = repo.CreateUser(second)
	require.NoError(t, err, "expected CreateUser to succeed")

	all, err := repo.GetAllUsers()
	require.NoError(t, err, "expected GetAllUsers to succeed")

	firstIndex, secondIndex := -1, -1
	for i, u := range all {
		switch u.Uuid {
		case first.Uuid:
			firstIndex = i
		case second.Uuid:
			secondIndex = i
		}
	}
	require.NotEqual(t, -1, firstIndex, "expected the oldest user to be present in GetAllUsers")
	require.NotEqual(t, -1, secondIndex, "expected the newest user to be present in GetAllUsers")
	assert.False(t, secondIndex >= firstIndex,
		"expected newer users to come first; newest-user at index %d, oldest-user at index %d", secondIndex, firstIndex)
}
