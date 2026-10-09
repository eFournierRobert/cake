package providers

import (
	"context"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"
	"uuid"

	"efournierrobert/cake-backend/internal/encryptor"
	"efournierrobert/cake-backend/internal/handlers/handler_errors"
	"efournierrobert/cake-backend/internal/handlers/models"
	"efournierrobert/cake-backend/internal/repository"
	providersRepo "efournierrobert/cake-backend/internal/repository/providers"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/mariadb"
)

// Each test seeds providers with a distinct name prefix, so tests
// never collide with each other and only clean up their own rows.
const (
	prefixList   = "svc-list-"
	prefixGet    = "svc-get-"
	prefixCreate = "svc-create-"
	prefixModify = "svc-modify-"
	prefixDelete = "svc-delete-"
	prefixTest   = "svc-test-"
)

// A 32-byte secret so aes.NewCipher accepts it and every encryption
// in the tests round-trips.
const testEncryptionKey = "test-only-encryption-secret-32b!"

var (
	testDB        *sqlx.DB
	testEncryptor *encryptor.Encryptor
)

// TestMain starts a fresh MariaDB container, points DATABASE_URL at
// it and opens the connection through repository.NewDbConnection so
// all pending migrations run, exactly like the application does.
// testDB holds that migrated connection and testEncryptor an
// encryptor built from a fixed test secret, for the whole test
// binary.
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

	os.Setenv("ENCRYPTION_SECRET", testEncryptionKey)
	testEncryptor, err = encryptor.New()
	if err != nil {
		log.Fatalf("test setup: failed to build the encryptor: %v", err)
	}

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

// service builds a providers service on the shared migrated test
// database.
func service() *Service {
	return New(testDB, testEncryptor)
}

// makeProvider inserts a provider directly through the database —
// including timestamps, which the repository only fills in through
// CreateProvider — and returns the row as found. A non-empty apiKey
// is stored encrypted, the way the service stores keys; an empty one
// stores a NULL api_key.
func makeProvider(t *testing.T, name, baseUrl, apiKey string) providersRepo.Provider {
	t.Helper()

	var p providersRepo.Provider
	p.Uuid = uuid.NewV4().String()

	var apiKeyArg any
	if apiKey != "" {
		blob, err := testEncryptor.Encrypt(apiKey)
		require.NoError(t, err, "failed to encrypt the test api key")
		apiKeyArg = blob
	}

	_, err := testDB.Exec(
		`INSERT INTO providers (uuid, name, base_url, api_key, created_at, updated_at)
		 VALUES (?, ?, ?, ?, NOW(6), NOW(6))`,
		p.Uuid, name, baseUrl, apiKeyArg,
	)
	require.NoError(t, err, "failed to insert provider %q", name)

	err = testDB.Get(&p, "SELECT * FROM providers WHERE uuid = ?", p.Uuid)
	require.NoError(t, err, "failed to look up inserted provider %q", name)

	return p
}

// deleteProviders removes the given providers by uuid, used by tests
// to clean up their own rows.
func deleteProviders(t *testing.T, ps ...providersRepo.Provider) {
	t.Helper()
	for _, p := range ps {
		_, err := testDB.Exec("DELETE FROM providers WHERE uuid = ?", p.Uuid)
		require.NoError(t, err, "failed to delete provider %q", p.Uuid)
	}
}

// insertModel inserts a model row referencing the provider directly
// through the test database, since no model repository exists yet.
// It returns the model row's id.
func insertModel(t *testing.T, provider providersRepo.Provider) int {
	t.Helper()

	var modelID int
	_, err := testDB.Exec(
		"INSERT INTO models (uuid, name, context_length, provider_model_id, provider_id, created_at) VALUES (?, ?, ?, ?, ?, NOW())",
		uuid.NewV4().String(), "gpt-test", 4096, "provider-model-id", provider.Id,
	)
	require.NoError(t, err, "failed to insert a model referencing the provider")
	err = testDB.QueryRow(
		"SELECT id FROM models WHERE provider_id = ? AND provider_model_id = ?", provider.Id, "provider-model-id",
	).Scan(&modelID)
	require.NoError(t, err, "failed to look up the inserted model")

	return modelID
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

// storedApiKey reads the raw api_key blob of the provider with the
// given uuid (nil when the column is NULL).
func storedApiKey(t *testing.T, providerUuid string) []byte {
	t.Helper()
	var key []byte
	err := testDB.QueryRow("SELECT api_key FROM providers WHERE uuid = ?", providerUuid).Scan(&key)
	require.NoError(t, err, "failed to read the stored api_key")
	return key
}

// index finds the position of the dto with the given uuid, or -1.
func index(all []models.ProviderDto, providerUuid string) int {
	for i, dto := range all {
		if dto.Uuid == providerUuid {
			return i
		}
	}
	return -1
}

// strPtr returns a pointer to s, for building ProviderUpdate api
// keys, whose nil-ness carries meaning.
func strPtr(s string) *string {
	return &s
}

// fakeProvider is a minimal stand-in for a provider /models endpoint.
// It answers every request with a fixed status code and records what
// it received, so tests can assert on the request the service sent.
type fakeProvider struct {
	url string

	mu            sync.Mutex
	method        string
	path          string
	authorization string
}

// newFakeProvider starts an httptest server answering every request
// with the given status code. It is closed when the test ends.
func newFakeProvider(t *testing.T, status int) *fakeProvider {
	t.Helper()

	fake := &fakeProvider{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fake.mu.Lock()
		fake.method = r.Method
		fake.path = r.URL.Path
		fake.authorization = r.Header.Get("Authorization")
		fake.mu.Unlock()
		w.WriteHeader(status)
	}))
	t.Cleanup(server.Close)
	fake.url = server.URL

	return fake
}

// snapshot returns what the fake provider recorded so far.
func (f *fakeProvider) snapshot() (method, path, authorization string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.method, f.path, f.authorization
}

// This is the first test of the binary, so it runs against the freshly
// migrated database: the table only holds what earlier tests left
// behind, and every test cleans up its own rows.
func TestGetAllProviders(t *testing.T) {
	svc := service()

	t.Run("returns an empty list on an empty providers table", func(t *testing.T) {
		all, err := svc.GetAllProviders()
		require.NoError(t, err, "expected GetAllProviders to succeed on an empty providers table")
		assert.Empty(t, all, "expected no providers before anything is created")
	})

	t.Run("returns every provider newest first with api key flags", func(t *testing.T) {
		oldest := makeProvider(t, prefixList+"oldest", "https://api.oldest.example.com", "oldest-key-123")
		// DATETIME(6) stores microseconds; the sleeps make sure the
		// three created_at values are strictly ordered.
		time.Sleep(10 * time.Millisecond)
		middle := makeProvider(t, prefixList+"middle", "https://api.middle.example.com", "")
		time.Sleep(10 * time.Millisecond)
		newest := makeProvider(t, prefixList+"newest", "https://api.newest.example.com", "newest-key-12")
		t.Cleanup(func() { deleteProviders(t, oldest, middle, newest) })

		all, err := svc.GetAllProviders()
		require.NoError(t, err, "expected GetAllProviders to succeed")

		oldestIdx := index(all, oldest.Uuid)
		middleIdx := index(all, middle.Uuid)
		newestIdx := index(all, newest.Uuid)
		require.NotEqual(t, -1, oldestIdx, "expected the oldest provider to be listed")
		require.NotEqual(t, -1, middleIdx, "expected the middle provider to be listed")
		require.NotEqual(t, -1, newestIdx, "expected the newest provider to be listed")
		assert.True(t, newestIdx < middleIdx, "expected newest (%d) to come before middle (%d)", newestIdx, middleIdx)
		assert.True(t, middleIdx < oldestIdx, "expected middle (%d) to come before oldest (%d)", middleIdx, oldestIdx)

		byUuid := map[string]models.ProviderDto{}
		for _, dto := range all {
			byUuid[dto.Uuid] = dto
		}
		assert.True(t, byUuid[oldest.Uuid].HasApiKey, "expected the oldest provider to report an api key")
		assert.False(t, byUuid[middle.Uuid].HasApiKey, "expected the middle provider to report no api key")
		assert.True(t, byUuid[newest.Uuid].HasApiKey, "expected the newest provider to report an api key")
		assert.Equal(t, prefixList+"newest", byUuid[newest.Uuid].Name)
	})
}

func TestGetProvider(t *testing.T) {
	svc := service()

	t.Run("returns the provider with its api key flag", func(t *testing.T) {
		provider := makeProvider(t, prefixGet+"keyed", "https://api.keyed.example.com", "get-key-12345")
		t.Cleanup(func() { deleteProviders(t, provider) })

		dto, err := svc.GetProvider(provider.Uuid)
		require.NoError(t, err, "expected GetProvider to succeed")

		assert.Equal(t, provider.Uuid, dto.Uuid)
		assert.Equal(t, prefixGet+"keyed", dto.Name)
		assert.Equal(t, "https://api.keyed.example.com", dto.BaseUrl)
		assert.True(t, dto.HasApiKey, "expected the stored api key to be reported")
		assert.Equal(t, provider.CreatedAt, dto.CreatedAt)
		assert.Equal(t, provider.UpdatedAt, dto.UpdatedAt)
	})

	t.Run("reports a provider without an api key", func(t *testing.T) {
		provider := makeProvider(t, prefixGet+"plain", "https://api.plain.example.com", "")
		t.Cleanup(func() { deleteProviders(t, provider) })

		dto, err := svc.GetProvider(provider.Uuid)
		require.NoError(t, err, "expected GetProvider to succeed")
		assert.False(t, dto.HasApiKey, "expected no api key to be reported")
	})

	t.Run("unknown uuid", func(t *testing.T) {
		_, err := svc.GetProvider(uuid.NewV4().String())
		assert.ErrorIs(t, err, handler_errors.ErrProviderNotFound)
	})

	t.Run("malformed uuid", func(t *testing.T) {
		_, err := svc.GetProvider("not-a-uuid")
		assert.ErrorIs(t, err, handler_errors.ErrInvalidRequest)
	})
}

func TestCreateProvider(t *testing.T) {
	svc := service()

	t.Run("creates a provider without an api key", func(t *testing.T) {
		created, err := svc.CreateProvider(models.ProviderCreate{
			Name:    prefixCreate + "plain",
			BaseUrl: "https://api.plain.example.com/",
		})
		require.NoError(t, err, "expected CreateProvider to succeed")
		t.Cleanup(func() { deleteProviders(t, providersRepo.Provider{Uuid: created.Uuid}) })

		_, err = uuid.Parse(created.Uuid)
		require.NoError(t, err, "expected the created uuid to be parseable")
		assert.Equal(t, prefixCreate+"plain", created.Name)
		assert.Equal(t, "https://api.plain.example.com", created.BaseUrl,
			"expected the trailing slash to be trimmed")
		assert.False(t, created.HasApiKey, "expected no api key to be reported")
		assert.False(t, created.CreatedAt.IsZero(), "expected created_at to be populated")
		assert.Nil(t, storedApiKey(t, created.Uuid), "expected api_key to be stored as NULL")

		fetched, err := svc.GetProvider(created.Uuid)
		require.NoError(t, err, "expected the created provider to round-trip through GetProvider")
		assert.Equal(t, prefixCreate+"plain", fetched.Name)
		assert.False(t, fetched.HasApiKey)
	})

	t.Run("stores the api key encrypted at rest", func(t *testing.T) {
		apiKey := "sk-super-secret-value"
		created, err := svc.CreateProvider(models.ProviderCreate{
			Name:    prefixCreate + "secret",
			BaseUrl: "https://api.secret.example.com",
			ApiKey:  apiKey,
		})
		require.NoError(t, err, "expected CreateProvider with an api key to succeed")
		t.Cleanup(func() { deleteProviders(t, providersRepo.Provider{Uuid: created.Uuid}) })

		assert.True(t, created.HasApiKey, "expected the dto to report an api key")

		blob := storedApiKey(t, created.Uuid)
		assert.NotEqual(t, apiKey, string(blob),
			"expected the api key to be stored encrypted, not as plaintext")
		decrypted, err := testEncryptor.Decrypt(blob)
		require.NoError(t, err, "expected the stored blob to decrypt with the test secret")
		assert.Equal(t, apiKey, string(decrypted))
	})

	t.Run("accepts absolute http and https base urls", func(t *testing.T) {
		for _, baseUrl := range []string{
			"https://api.example.com",
			"http://127.0.0.1:8080",
			"https://api.example.com/v1",
		} {
			created, err := svc.CreateProvider(models.ProviderCreate{
				Name:    prefixCreate + "valid",
				BaseUrl: baseUrl,
			})
			require.NoError(t, err, "expected %q to be accepted", baseUrl)
			t.Cleanup(func() { deleteProviders(t, providersRepo.Provider{Uuid: created.Uuid}) })

			assert.Equal(t, baseUrl, created.BaseUrl, "expected the base url to be stored as given")
		}
	})

	t.Run("rejects invalid base urls", func(t *testing.T) {
		for _, baseUrl := range []string{
			"",
			"api.example.com",
			"//protocol-relative.example.com",
			"ftp://files.example.com",
			"http://",
		} {
			_, err := svc.CreateProvider(models.ProviderCreate{
				Name:    prefixCreate + "invalid",
				BaseUrl: baseUrl,
			})
			assert.ErrorIs(t, err, handler_errors.ErrInvalidRequest, "expected %q to be rejected", baseUrl)
		}

		assert.Zero(t, countRows(t, "SELECT COUNT(*) FROM providers WHERE name = ?", prefixCreate+"invalid"),
			"expected no provider to be inserted for an invalid base url")
	})
}

func TestModifyProvider(t *testing.T) {
	svc := service()

	t.Run("updates the provided fields", func(t *testing.T) {
		provider := makeProvider(t, prefixModify+"fields", "https://api.original.example.com", "old-key-12345")
		t.Cleanup(func() { deleteProviders(t, provider) })

		// DATETIME(6) stores microseconds, so updated_at is guaranteed
		// to move forward as soon as any measurable time passes; the
		// sleep only guards against the two measurements landing on
		// the same microsecond.
		time.Sleep(10 * time.Millisecond)

		dto, err := svc.ModifyProvider(provider.Uuid, models.ProviderUpdate{
			Name:    prefixModify + "renamed",
			BaseUrl: "https://api.renamed.example.com/",
		})
		require.NoError(t, err, "expected ModifyProvider to succeed")

		assert.Equal(t, provider.Uuid, dto.Uuid)
		assert.Equal(t, prefixModify+"renamed", dto.Name)
		assert.Equal(t, "https://api.renamed.example.com", dto.BaseUrl,
			"expected the trailing slash to be trimmed")
		assert.Equal(t, provider.CreatedAt, dto.CreatedAt, "expected created_at to be unchanged")
		assert.True(t, dto.UpdatedAt.After(provider.UpdatedAt),
			"expected updated_at (%v) to move forward from (%v)", dto.UpdatedAt, provider.UpdatedAt)
		assert.True(t, dto.HasApiKey, "expected the untouched api key to be preserved")
	})

	t.Run("leaves empty fields unchanged", func(t *testing.T) {
		provider := makeProvider(t, prefixModify+"partial", "https://api.partial.example.com", "keep-key-1234")
		blobBefore := storedApiKey(t, provider.Uuid)
		t.Cleanup(func() { deleteProviders(t, provider) })

		// A nil ApiKey must keep the existing key; only a non-nil
		// value (even an empty string) touches it.
		dto, err := svc.ModifyProvider(provider.Uuid, models.ProviderUpdate{Name: prefixModify + "renamed-partial"})
		require.NoError(t, err, "expected ModifyProvider to succeed")

		assert.Equal(t, prefixModify+"renamed-partial", dto.Name)
		assert.Equal(t, "https://api.partial.example.com", dto.BaseUrl, "expected base_url to be left unchanged")
		assert.True(t, dto.HasApiKey, "expected the api key to be kept")
		assert.Equal(t, blobBefore, storedApiKey(t, provider.Uuid),
			"expected the stored api key to be left unchanged")
	})

	t.Run("all-empty update does not modify the provider", func(t *testing.T) {
		provider := makeProvider(t, prefixModify+"empty", "https://api.empty.example.com", "empty-key-1234")
		t.Cleanup(func() { deleteProviders(t, provider) })

		dto, err := svc.ModifyProvider(provider.Uuid, models.ProviderUpdate{})
		require.NoError(t, err, "expected ModifyProvider to succeed")

		assert.Equal(t, prefixModify+"empty", dto.Name)
		assert.Equal(t, "https://api.empty.example.com", dto.BaseUrl)
		assert.True(t, dto.HasApiKey)
	})

	t.Run("empty api key clears the stored key", func(t *testing.T) {
		provider := makeProvider(t, prefixModify+"clearkey", "https://api.clear.example.com", "doomed-key-123")
		t.Cleanup(func() { deleteProviders(t, provider) })

		dto, err := svc.ModifyProvider(provider.Uuid, models.ProviderUpdate{ApiKey: strPtr("")})
		require.NoError(t, err, "expected ModifyProvider to succeed when clearing the api key")

		assert.False(t, dto.HasApiKey, "expected the dto to report no api key")
		assert.Nil(t, storedApiKey(t, provider.Uuid), "expected api_key to be cleared in the database")
		assert.Equal(t, "https://api.clear.example.com", dto.BaseUrl, "expected base_url to be left unchanged")
	})

	t.Run("replacing the api key stores the new key encrypted", func(t *testing.T) {
		provider := makeProvider(t, prefixModify+"rekey", "https://api.rekey.example.com", "old-key-123456")
		blobBefore := storedApiKey(t, provider.Uuid)
		t.Cleanup(func() { deleteProviders(t, provider) })

		dto, err := svc.ModifyProvider(provider.Uuid, models.ProviderUpdate{ApiKey: strPtr("new-key-654321")})
		require.NoError(t, err, "expected ModifyProvider to succeed when replacing the api key")

		assert.True(t, dto.HasApiKey)
		blobAfter := storedApiKey(t, provider.Uuid)
		assert.NotEqual(t, blobBefore, blobAfter, "expected the stored api key to change")
		decrypted, err := testEncryptor.Decrypt(blobAfter)
		require.NoError(t, err, "expected the stored blob to decrypt with the test secret")
		assert.Equal(t, "new-key-654321", string(decrypted))
	})

	t.Run("rejects an invalid base url", func(t *testing.T) {
		provider := makeProvider(t, prefixModify+"badurl", "https://api.badurl.example.com", "")
		t.Cleanup(func() { deleteProviders(t, provider) })

		_, err := svc.ModifyProvider(provider.Uuid, models.ProviderUpdate{BaseUrl: "api.example.com"})
		assert.ErrorIs(t, err, handler_errors.ErrInvalidRequest)

		unchanged, err := svc.GetProvider(provider.Uuid)
		require.NoError(t, err, "expected the provider to be fetchable after the rejected update")
		assert.Equal(t, "https://api.badurl.example.com", unchanged.BaseUrl,
			"expected base_url to be left unchanged")
	})

	t.Run("unknown uuid", func(t *testing.T) {
		_, err := svc.ModifyProvider(uuid.NewV4().String(), models.ProviderUpdate{Name: "X"})
		assert.ErrorIs(t, err, handler_errors.ErrProviderNotFound)
	})

	t.Run("malformed uuid", func(t *testing.T) {
		_, err := svc.ModifyProvider("nonsense", models.ProviderUpdate{Name: "X"})
		assert.ErrorIs(t, err, handler_errors.ErrInvalidRequest)
	})
}

func TestDeleteProvider(t *testing.T) {
	svc := service()

	t.Run("deletes the provider and its models", func(t *testing.T) {
		provider := makeProvider(t, prefixDelete+"cascade", "https://api.cascade.example.com", "delete-key-123")
		modelID := insertModel(t, provider)

		require.NoError(t, svc.DeleteProvider(provider.Uuid), "expected DeleteProvider to succeed")

		assert.Zero(t, countRows(t, "SELECT COUNT(*) FROM providers WHERE uuid = ?", provider.Uuid),
			"expected the provider to be deleted")
		assert.Zero(t, countRows(t, "SELECT COUNT(*) FROM models WHERE id = ?", modelID),
			"expected the referencing models to be cascade deleted")
	})

	t.Run("unknown uuid", func(t *testing.T) {
		err := svc.DeleteProvider(uuid.NewV4().String())
		assert.ErrorIs(t, err, handler_errors.ErrProviderNotFound)
	})

	t.Run("malformed uuid", func(t *testing.T) {
		err := svc.DeleteProvider("nonsense")
		assert.ErrorIs(t, err, handler_errors.ErrInvalidRequest)
	})
}

func TestTestProvider(t *testing.T) {
	svc := service()

	t.Run("succeeds on a 2xx response", func(t *testing.T) {
		fake := newFakeProvider(t, http.StatusOK)
		provider := makeProvider(t, prefixTest+"ok", fake.url, "")
		t.Cleanup(func() { deleteProviders(t, provider) })

		res, err := svc.TestProvider(provider.Uuid)
		require.NoError(t, err, "expected TestProvider to succeed")

		assert.True(t, res.Success)
		assert.Empty(t, res.Error)
		require.NotNil(t, res.LatencyMs, "expected the latency to be measured")
		assert.GreaterOrEqual(t, *res.LatencyMs, int64(0))

		method, path, authorization := fake.snapshot()
		assert.Equal(t, http.MethodGet, method, "expected the provider to be probed with GET")
		assert.Equal(t, "/models", path, "expected the provider to be probed on /models")
		assert.Empty(t, authorization, "expected no authorization header without a stored api key")
	})

	t.Run("sends the stored api key as a bearer token", func(t *testing.T) {
		fake := newFakeProvider(t, http.StatusOK)
		provider := makeProvider(t, prefixTest+"bearer", fake.url, "sk-live-12345")
		t.Cleanup(func() { deleteProviders(t, provider) })

		res, err := svc.TestProvider(provider.Uuid)
		require.NoError(t, err, "expected TestProvider to succeed")
		assert.True(t, res.Success)

		_, _, authorization := fake.snapshot()
		assert.Equal(t, "Bearer sk-live-12345", authorization,
			"expected the decrypted api key to be sent as a bearer token")
	})

	t.Run("reports the http status on a non-2xx response", func(t *testing.T) {
		fake := newFakeProvider(t, http.StatusNotFound)
		provider := makeProvider(t, prefixTest+"status", fake.url, "")
		t.Cleanup(func() { deleteProviders(t, provider) })

		res, err := svc.TestProvider(provider.Uuid)
		require.NoError(t, err, "expected TestProvider to report the failure through the response, not an error")

		assert.False(t, res.Success)
		assert.Equal(t, "provider returned HTTP 404", res.Error)
		require.NotNil(t, res.LatencyMs, "expected the latency to be measured even on failure")
	})

	t.Run("reports a failure when the provider is unreachable", func(t *testing.T) {
		// A closed server leaves an address that refuses connections,
		// so the request fails fast.
		server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
		unreachableUrl := server.URL
		server.Close()

		provider := makeProvider(t, prefixTest+"down", unreachableUrl, "")
		t.Cleanup(func() { deleteProviders(t, provider) })

		res, err := svc.TestProvider(provider.Uuid)
		require.NoError(t, err, "expected TestProvider to report the failure through the response, not an error")

		assert.False(t, res.Success)
		assert.Equal(t, "error happened when testing the provider", res.Error)
		require.NotNil(t, res.LatencyMs, "expected the latency to be measured even on failure")
	})

	t.Run("reports an unexpected error when the stored api key cannot be decrypted", func(t *testing.T) {
		provider := makeProvider(t, prefixTest+"corrupt", "https://api.corrupt.example.com", "")
		t.Cleanup(func() { deleteProviders(t, provider) })

		// Overwrite api_key with bytes that are not a valid ciphertext
		// for the test secret.
		_, err := testDB.Exec("UPDATE providers SET api_key = ? WHERE uuid = ?",
			[]byte("not-a-valid-ciphertext"), provider.Uuid)
		require.NoError(t, err, "failed to corrupt the stored api key")

		_, err = svc.TestProvider(provider.Uuid)
		assert.ErrorIs(t, err, handler_errors.ErrUnexpectedError)
	})

	t.Run("unknown uuid", func(t *testing.T) {
		_, err := svc.TestProvider(uuid.NewV4().String())
		assert.ErrorIs(t, err, handler_errors.ErrProviderNotFound)
	})

	t.Run("malformed uuid", func(t *testing.T) {
		_, err := svc.TestProvider("nonsense")
		assert.ErrorIs(t, err, handler_errors.ErrInvalidRequest)
	})
}
