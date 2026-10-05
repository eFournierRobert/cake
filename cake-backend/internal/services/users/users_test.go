package users

import (
	"context"
	"log"
	"os"
	"strings"
	"testing"
	"uuid"

	"efournierrobert/cake-backend/internal/handlers/handler_errors"
	userHandler "efournierrobert/cake-backend/internal/handlers/users"
	"efournierrobert/cake-backend/internal/repository"
	userRepo "efournierrobert/cake-backend/internal/repository/users"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/mariadb"
	"golang.org/x/crypto/bcrypt"
)

// Each test seeds users with a distinct username prefix, so tests never
// collide with each other and only clean up their own rows.
const (
	prefixGet    = "svc-get-"
	prefixModify = "svc-modify-"
	prefixAdmin  = "svc-admin-"
	prefixChange = "svc-change-"
	prefixList   = "svc-list-"
	prefixCreate = "svc-create-"
	prefixDelete = "svc-delete-"
)

// A default, valid-length password for tests that do not exercise the
// password length rules.
const testPassword = "sup3r-s3cure-pass!"

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

// service builds a users service on the shared migrated test database.
func service() *Service {
	return New(testDB)
}

// roleID looks up the id of a seeded user role by name.
func roleID(t *testing.T, name string) int {
	t.Helper()
	var id int
	err := testDB.QueryRow("SELECT id FROM user_roles WHERE name = ?", name).Scan(&id)
	require.NoError(t, err, "expected the seeded %q role to exist", name)
	return id
}

// makeUser inserts a user directly through the database — including
// timestamps, which the repository only fills in through
// CreateUser — and returns the row as found.
func makeUser(t *testing.T, username, role, firstName, lastName, password string) userRepo.User {
	t.Helper()

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	require.NoError(t, err, "failed to hash the test password")

	var u userRepo.User
	u.Uuid = uuid.NewV4().String()
	_, err = testDB.Exec(
		`INSERT INTO users (uuid, username, password_hash, first_name, last_name, created_at, updated_at, role_id)
		 VALUES (?, ?, ?, ?, ?, NOW(6), NOW(6), ?)`,
		u.Uuid, username, hash, firstName, lastName, roleID(t, role),
	)
	require.NoError(t, err, "failed to insert user %q", username)

	err = testDB.Get(&u, "SELECT * FROM users WHERE uuid = ?", u.Uuid)
	require.NoError(t, err, "failed to look up inserted user %q", username)

	return u
}

// deleteUsers removes the given users by uuid, used by tests to clean
// up their own rows.
func deleteUsers(t *testing.T, users ...userRepo.User) {
	t.Helper()
	for _, u := range users {
		_, err := testDB.Exec("DELETE FROM users WHERE uuid = ?", u.Uuid)
		require.NoError(t, err, "failed to delete user %q", u.Uuid)
	}
}

// insertOwnedConversation inserts a space owned by the user and a
// conversation in it, directly through the test database, since no
// conversation or space repository exists yet. It returns the
// conversation id.
func insertOwnedConversation(t *testing.T, owner userRepo.User) int {
	t.Helper()

	var spaceID, conversationID int

	_, err := testDB.Exec(
		"INSERT INTO spaces (uuid, name, owner_id, created_at, updated_at) VALUES (?, ?, ?, NOW(), NOW())",
		uuid.NewV4().String(), "owned space", owner.Id,
	)
	require.NoError(t, err, "failed to insert a space owned by the user")
	err = testDB.QueryRow(
		"SELECT id FROM spaces WHERE owner_id = ?", owner.Id,
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

// storedPasswordHash reads the current password_hash of the user with
// the given uuid.
func storedPasswordHash(t *testing.T, userUuid string) []byte {
	t.Helper()
	var hash []byte
	err := testDB.QueryRow("SELECT password_hash FROM users WHERE uuid = ?", userUuid).Scan(&hash)
	require.NoError(t, err, "failed to read the stored password hash")
	return hash
}

// index finds the position of the dto with the given username, or -1.
func index(all []userHandler.UserDto, username string) int {
	for i, dto := range all {
		if dto.Username == username {
			return i
		}
	}
	return -1
}

func TestGetUser(t *testing.T) {
	svc := service()

	t.Run("returns the user with its role resolved", func(t *testing.T) {
		user := makeUser(t, prefixGet+"ada", "user", "Ada", "Lovelace", testPassword)
		t.Cleanup(func() { deleteUsers(t, user) })

		dto, err := svc.GetUser(user.Uuid)
		require.NoError(t, err, "expected GetUser to succeed")

		assert.Equal(t, user.Uuid, dto.Uuid)
		assert.Equal(t, prefixGet+"ada", dto.Username)
		assert.Equal(t, "user", dto.Role)
		assert.Equal(t, "Ada", dto.FirstName)
		assert.Equal(t, "Lovelace", dto.LastName)
		assert.Equal(t, user.CreatedAt, dto.CreatedAt)
		assert.Equal(t, user.UpdatedAt, dto.LastUpdated)
	})

	t.Run("unknown uuid", func(t *testing.T) {
		_, err := svc.GetUser(uuid.NewV4().String())
		assert.ErrorIs(t, err, handler_errors.ErrUserDoesNotExist)
	})

	t.Run("malformed uuid", func(t *testing.T) {
		_, err := svc.GetUser("not-a-uuid")
		assert.ErrorIs(t, err, handler_errors.ErrInvalidRequest)
	})
}

func TestModifyUser(t *testing.T) {
	svc := service()

	t.Run("updates every provided field", func(t *testing.T) {
		user := makeUser(t, prefixModify+"ada", "user", "Ada", "Lovelace", testPassword)
		t.Cleanup(func() { deleteUsers(t, user) })

		dto, err := svc.ModifyUser(user.Uuid, userHandler.UserUpdate{
			FirstName: "Augusta",
			LastName:  "King",
			Username:  prefixModify + "augusta",
		})
		require.NoError(t, err, "expected ModifyUser to succeed")

		assert.Equal(t, "Augusta", dto.FirstName)
		assert.Equal(t, "King", dto.LastName)
		assert.Equal(t, prefixModify+"augusta", dto.Username)
		assert.False(t, dto.LastUpdated.Before(user.UpdatedAt),
			"expected updated_at (%v) to move forward from (%v)", dto.LastUpdated, user.UpdatedAt)
	})

	t.Run("leaves empty fields unchanged", func(t *testing.T) {
		user := makeUser(t, prefixModify+"grace", "user", "Grace", "Hopper", testPassword)
		t.Cleanup(func() { deleteUsers(t, user) })

		dto, err := svc.ModifyUser(user.Uuid, userHandler.UserUpdate{FirstName: "Grace B"})
		require.NoError(t, err, "expected ModifyUser to succeed")

		assert.Equal(t, "Grace B", dto.FirstName)
		assert.Equal(t, "Hopper", dto.LastName, "expected last_name to be left unchanged")
		assert.Equal(t, prefixModify+"grace", dto.Username, "expected username to be left unchanged")
	})

	t.Run("all-empty update does not modify the user", func(t *testing.T) {
		user := makeUser(t, prefixModify+"empty", "user", "Ada", "Lovelace", testPassword)
		t.Cleanup(func() { deleteUsers(t, user) })

		dto, err := svc.ModifyUser(user.Uuid, userHandler.UserUpdate{})
		require.NoError(t, err, "expected ModifyUser to succeed")

		assert.Equal(t, "Ada", dto.FirstName)
		assert.Equal(t, "Lovelace", dto.LastName)
		assert.Equal(t, prefixModify+"empty", dto.Username)
	})

	t.Run("unknown uuid", func(t *testing.T) {
		_, err := svc.ModifyUser(uuid.NewV4().String(), userHandler.UserUpdate{FirstName: "X"})
		assert.ErrorIs(t, err, handler_errors.ErrUserDoesNotExist)
	})

	t.Run("malformed uuid", func(t *testing.T) {
		_, err := svc.ModifyUser("nonsense", userHandler.UserUpdate{FirstName: "X"})
		assert.ErrorIs(t, err, handler_errors.ErrInvalidRequest)
	})

	t.Run("duplicate username", func(t *testing.T) {
		other := makeUser(t, prefixModify+"taken", "user", "Other", "User", testPassword)
		mine := makeUser(t, prefixModify+"mine", "user", "Mine", "User", testPassword)
		t.Cleanup(func() { deleteUsers(t, other, mine) })

		_, err := svc.ModifyUser(mine.Uuid, userHandler.UserUpdate{Username: prefixModify + "taken"})
		assert.ErrorIs(t, err, handler_errors.ErrResourceConflict)
	})
}

func TestAdminModifyUser(t *testing.T) {
	svc := service()

	t.Run("changes the role and the provided fields", func(t *testing.T) {
		user := makeUser(t, prefixAdmin+"ada", "user", "Ada", "Lovelace", testPassword)
		t.Cleanup(func() { deleteUsers(t, user) })

		dto, err := svc.AdminModifyUser(user.Uuid, userHandler.AdminUserUpdate{
			Role:      "admin",
			FirstName: "Augusta",
		})
		require.NoError(t, err, "expected AdminModifyUser to succeed")

		assert.Equal(t, "admin", dto.Role)
		assert.Equal(t, "Augusta", dto.FirstName)
		assert.Equal(t, "Lovelace", dto.LastName, "expected last_name to be left unchanged")
	})

	t.Run("empty role leaves the role unchanged", func(t *testing.T) {
		user := makeUser(t, prefixAdmin+"grace", "user", "Grace", "Hopper", testPassword)
		t.Cleanup(func() { deleteUsers(t, user) })

		dto, err := svc.AdminModifyUser(user.Uuid, userHandler.AdminUserUpdate{Username: prefixAdmin + "grace2"})
		require.NoError(t, err, "expected AdminModifyUser to succeed when the role is left empty")

		assert.Equal(t, "user", dto.Role, "expected the role to be left unchanged")
		assert.Equal(t, prefixAdmin+"grace2", dto.Username)
	})

	t.Run("unknown uuid", func(t *testing.T) {
		_, err := svc.AdminModifyUser(uuid.NewV4().String(), userHandler.AdminUserUpdate{Role: "admin"})
		assert.ErrorIs(t, err, handler_errors.ErrUserDoesNotExist)
	})

	t.Run("malformed uuid", func(t *testing.T) {
		_, err := svc.AdminModifyUser("nope", userHandler.AdminUserUpdate{Role: "admin"})
		assert.ErrorIs(t, err, handler_errors.ErrInvalidRequest)
	})

	t.Run("unknown role name", func(t *testing.T) {
		user := makeUser(t, prefixAdmin+"role", "user", "Role", "Error", testPassword)
		t.Cleanup(func() { deleteUsers(t, user) })

		_, err := svc.AdminModifyUser(user.Uuid, userHandler.AdminUserUpdate{Role: "superadmin"})
		assert.ErrorIs(t, err, handler_errors.ErrRoleNotFound)
	})

	t.Run("duplicate username", func(t *testing.T) {
		other := makeUser(t, prefixAdmin+"taken", "user", "Other", "User", testPassword)
		mine := makeUser(t, prefixAdmin+"mine", "user", "Mine", "User", testPassword)
		t.Cleanup(func() { deleteUsers(t, other, mine) })

		_, err := svc.AdminModifyUser(mine.Uuid, userHandler.AdminUserUpdate{Role: "admin", Username: prefixAdmin + "taken"})
		assert.ErrorIs(t, err, handler_errors.ErrResourceConflict)
	})
}

func TestChangePassword(t *testing.T) {
	svc := service()

	t.Run("replaces the stored password", func(t *testing.T) {
		user := makeUser(t, prefixChange+"pw", "user", "Pw", "Change", "old-old-old-pass")
		t.Cleanup(func() { deleteUsers(t, user) })

		newPassword := "new-new-new-pass"
		require.NoError(t, svc.ChangePassword(user.Uuid, newPassword), "expected ChangePassword to succeed")

		hash := storedPasswordHash(t, user.Uuid)
		assert.NoError(t, bcrypt.CompareHashAndPassword(hash, []byte(newPassword)),
			"expected the new password to match the stored hash")
		assert.Error(t, bcrypt.CompareHashAndPassword(hash, []byte("old-old-old-pass")),
			"expected the old password to no longer match the stored hash")
	})

	t.Run("too short", func(t *testing.T) {
		user := makeUser(t, prefixChange+"short", "user", "Pw", "Change", testPassword)
		t.Cleanup(func() { deleteUsers(t, user) })

		err := svc.ChangePassword(user.Uuid, "short")
		assert.ErrorIs(t, err, handler_errors.ErrInvalidPassword)
	})

	t.Run("too long", func(t *testing.T) {
		user := makeUser(t, prefixChange+"long", "user", "Pw", "Change", testPassword)
		t.Cleanup(func() { deleteUsers(t, user) })

		err := svc.ChangePassword(user.Uuid, strings.Repeat("a", 73))
		assert.ErrorIs(t, err, handler_errors.ErrInvalidPassword)
	})

	t.Run("unknown uuid", func(t *testing.T) {
		err := svc.ChangePassword(uuid.NewV4().String(), "valid-new-pass-1")
		assert.ErrorIs(t, err, handler_errors.ErrUserDoesNotExist)
	})

	t.Run("malformed uuid", func(t *testing.T) {
		err := svc.ChangePassword("nonsense", "valid-new-pass-1")
		assert.ErrorIs(t, err, handler_errors.ErrInvalidRequest)
	})
}

func TestGetAllUsers(t *testing.T) {
	svc := service()

	t.Run("returns every user newest first with resolved roles", func(t *testing.T) {
		oldest := makeUser(t, prefixList+"oldest", "user", "Ada", "Lovelace", testPassword)
		middle := makeUser(t, prefixList+"second", "admin", "Grace", "Hopper", testPassword)
		newest := makeUser(t, prefixList+"newest", "user", "Alan", "Turing", testPassword)
		t.Cleanup(func() { deleteUsers(t, oldest, middle, newest) })

		all, err := svc.GetAllUsers()
		require.NoError(t, err, "expected GetAllUsers to succeed")

		oldestIdx := index(all, oldest.Username)
		middleIdx := index(all, middle.Username)
		newestIdx := index(all, newest.Username)
		require.NotEqual(t, -1, oldestIdx, "expected the oldest user to be listed")
		require.NotEqual(t, -1, middleIdx, "expected the middle user to be listed")
		require.NotEqual(t, -1, newestIdx, "expected the newest user to be listed")
		assert.True(t, newestIdx < middleIdx, "expected newest (%d) to come before middle (%d)", newestIdx, middleIdx)
		assert.True(t, middleIdx < oldestIdx, "expected middle (%d) to come before oldest (%d)", middleIdx, oldestIdx)

		byUsername := map[string]userHandler.UserDto{}
		for _, dto := range all {
			byUsername[dto.Username] = dto
		}
		assert.Equal(t, "user", byUsername[oldest.Username].Role)
		assert.Equal(t, "admin", byUsername[middle.Username].Role)
		assert.Equal(t, "Alan", byUsername[newest.Username].FirstName)
	})

	t.Run("empty database", func(t *testing.T) {
		// A fresh test database contains only seeded roles, never
		// users, so GetAllUsers must return an empty list, not an
		// error.
		all, err := svc.GetAllUsers()
		require.NoError(t, err, "expected GetAllUsers to succeed on an empty users table")
		assert.Empty(t, all)
	})
}

func TestCreateUser(t *testing.T) {
	svc := service()

	t.Run("creates a user", func(t *testing.T) {
		created, err := svc.CreateUser(userHandler.UserCreate{
			Username:  prefixCreate + "ada",
			Password:  testPassword,
			Role:      "user",
			FirstName: "Ada",
			LastName:  "Lovelace",
		})
		require.NoError(t, err, "expected CreateUser to succeed")
		t.Cleanup(func() { deleteUsers(t, userRepo.User{Uuid: created.Uuid}) })

		_, err = uuid.Parse(created.Uuid)
		require.NoError(t, err, "expected the created uuid to be parseable")
		assert.Equal(t, prefixCreate+"ada", created.Username)
		assert.Equal(t, "user", created.Role)
		assert.Equal(t, "Ada", created.FirstName)
		assert.Equal(t, "Lovelace", created.LastName)
		assert.False(t, created.CreatedAt.IsZero(), "expected created_at to be populated")

		fetched, err := svc.GetUser(created.Uuid)
		require.NoError(t, err, "expected the created user to round-trip through GetUser")
		assert.Equal(t, prefixCreate+"ada", fetched.Username)
		assert.Equal(t, "user", fetched.Role)
	})

	t.Run("creates an admin", func(t *testing.T) {
		created, err := svc.CreateUser(userHandler.UserCreate{
			Username: prefixCreate + "admin",
			Password: testPassword,
			Role:     "admin",
		})
		require.NoError(t, err, "expected CreateUser with an admin role to succeed")
		t.Cleanup(func() { deleteUsers(t, userRepo.User{Uuid: created.Uuid}) })

		assert.Equal(t, "admin", created.Role, "expected the role name to be resolved from the user_roles table")
	})

	t.Run("password too short", func(t *testing.T) {
		_, err := svc.CreateUser(userHandler.UserCreate{
			Username: prefixCreate + "short",
			Password: "short",
			Role:     "user",
		})
		assert.ErrorIs(t, err, handler_errors.ErrInvalidPassword)
	})

	t.Run("password too long", func(t *testing.T) {
		_, err := svc.CreateUser(userHandler.UserCreate{
			Username: prefixCreate + "long",
			Password: strings.Repeat("a", 73),
			Role:     "user",
		})
		assert.ErrorIs(t, err, handler_errors.ErrInvalidPassword)
	})

	t.Run("unknown role", func(t *testing.T) {
		_, err := svc.CreateUser(userHandler.UserCreate{
			Username: prefixCreate + "norole",
			Password: testPassword,
			Role:     "superadmin",
		})
		assert.ErrorIs(t, err, handler_errors.ErrRoleNotFound)
	})

	t.Run("duplicate username", func(t *testing.T) {
		first, err := svc.CreateUser(userHandler.UserCreate{
			Username: prefixCreate + "twin",
			Password: testPassword,
			Role:     "user",
		})
		require.NoError(t, err, "expected the first user to be created")
		t.Cleanup(func() { deleteUsers(t, userRepo.User{Uuid: first.Uuid}) })

		_, err = svc.CreateUser(userHandler.UserCreate{
			Username: prefixCreate + "twin",
			Password: testPassword,
			Role:     "admin",
		})
		assert.ErrorIs(t, err, handler_errors.ErrResourceConflict)
	})
}

func TestDeleteUser(t *testing.T) {
	svc := service()

	t.Run("deletes the user and what they own", func(t *testing.T) {
		user := makeUser(t, prefixDelete+"ada", "user", "Ada", "Lovelace", testPassword)
		conversationID := insertOwnedConversation(t, user)

		require.NoError(t, svc.DeleteUser(user.Uuid), "expected DeleteUser to succeed")

		assert.Equal(t, 0, countRows(t, "SELECT COUNT(*) FROM users WHERE uuid = ?", user.Uuid),
			"expected the user to be deleted")
		assert.Equal(t, 0, countRows(t, "SELECT COUNT(*) FROM spaces WHERE owner_id = ?", user.Id),
			"expected the owned spaces to be cascade deleted")
		assert.Equal(t, 0, countRows(t, "SELECT COUNT(*) FROM conversations WHERE id = ?", conversationID),
			"expected the owned conversations to be cascade deleted")
	})

	t.Run("unknown uuid", func(t *testing.T) {
		err := svc.DeleteUser(uuid.NewV4().String())
		assert.ErrorIs(t, err, handler_errors.ErrUserDoesNotExist)
	})

	t.Run("malformed uuid", func(t *testing.T) {
		err := svc.DeleteUser("nonsense")
		assert.ErrorIs(t, err, handler_errors.ErrInvalidRequest)
	})
}
