package roles

import (
	"context"
	"log"
	"os"
	"testing"

	"efournierrobert/cake-backend/internal/repository"
	"efournierrobert/cake-backend/internal/repository/repo_errors"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/mariadb"
)

// testUserRoleUUID and testAdminRoleUUID are the uuids of the seeded
// user roles from the initial migration.
const (
	testUserRoleUUID  = "15afe83a-fd92-4d66-8f22-5a3bedbb53e1"
	testAdminRoleUUID = "5a18559d-9d20-4251-8a72-b36efbaff514"
)

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

func TestGetRoleByName(t *testing.T) {
	repo := New(testDB)

	t.Run("finds the seeded user role", func(t *testing.T) {
		role, err := repo.GetRoleByName("user")
		require.NoError(t, err, "expected the seeded user role to be found")

		assert.Positive(t, role.Id, "expected a positive auto-generated id")
		assert.Equal(t, testUserRoleUUID, role.Uuid)
		assert.Equal(t, "user", role.Name)
	})

	t.Run("finds the seeded admin role", func(t *testing.T) {
		role, err := repo.GetRoleByName("admin")
		require.NoError(t, err, "expected the seeded admin role to be found")

		assert.Positive(t, role.Id, "expected a positive auto-generated id")
		assert.Equal(t, testAdminRoleUUID, role.Uuid)
		assert.Equal(t, "admin", role.Name)
	})

	t.Run("unknown name", func(t *testing.T) {
		role, err := repo.GetRoleByName("superadmin")
		assert.ErrorIs(t, err, &repo_errors.RoleNotFound{},
			"expected *repo_errors.RoleNotFound for an unknown role name")
		assert.Zero(t, role, "expected the zero Role to be returned on error")
	})
}

func TestGetRoleById(t *testing.T) {
	repo := New(testDB)

	t.Run("finds the seeded user role by id", func(t *testing.T) {
		var id int
		err := testDB.QueryRow(
			"SELECT id FROM user_roles WHERE uuid = ?", testUserRoleUUID,
		).Scan(&id)
		require.NoError(t, err, "expected the seeded user role to exist")

		role, err := repo.GetRoleById(id)
		require.NoError(t, err, "expected the role to be found by id")

		assert.Equal(t, id, role.Id)
		assert.Equal(t, testUserRoleUUID, role.Uuid)
		assert.Equal(t, "user", role.Name)
	})

	t.Run("finds the seeded admin role by id", func(t *testing.T) {
		var id int
		err := testDB.QueryRow(
			"SELECT id FROM user_roles WHERE uuid = ?", testAdminRoleUUID,
		).Scan(&id)
		require.NoError(t, err, "expected the seeded admin role to exist")

		role, err := repo.GetRoleById(id)
		require.NoError(t, err, "expected the role to be found by id")

		assert.Equal(t, id, role.Id)
		assert.Equal(t, testAdminRoleUUID, role.Uuid)
		assert.Equal(t, "admin", role.Name)
	})

	t.Run("unknown id", func(t *testing.T) {
		role, err := repo.GetRoleById(999999)
		assert.ErrorIs(t, err, &repo_errors.RoleNotFound{},
			"expected *repo_errors.RoleNotFound for an unknown role id")
		assert.Zero(t, role, "expected the zero Role to be returned on error")
	})
}
