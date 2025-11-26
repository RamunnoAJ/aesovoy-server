package store

import (
	"crypto/sha256"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPasswordMatches(t *testing.T) {
	pp := &Password{}
	require.NoError(t, pp.Set("s3cret-Ok"))

	tests := []struct {
		name    string
		input   string
		wantOK  bool
		wantErr bool
	}{
		{"matches", "s3cret-Ok", true, false},
		{"doesn't match", "bad-one", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, err := pp.Matches(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantOK, ok)
		})
	}
}

func TestUserStore_Create(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	us := NewPostgresUserStore(db)

	u1 := &User{Username: "test", Email: "test@example.com", Role: "employee"}
	require.NoError(t, u1.PasswordHash.Set("password"))

	tests := []struct {
		name    string
		user    *User
		wantErr bool
	}{
		{
			name:    "create valid user",
			user:    u1,
			wantErr: false,
		},
		{
			name: "create duplicate user",
			user: &User{
				Username: "test", // Duplicate username
				Email:    "another@example.com",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := us.CreateUser(tt.user)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotZero(t, tt.user.ID)
			}
		})
	}
}

func TestUserStore_GetUpdate(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	us := NewPostgresUserStore(db)

	u := &User{Username: "me", Email: "me@example.com", Role: "administrator"}
	require.NoError(t, u.PasswordHash.Set("securepassword"))
	require.NoError(t, us.CreateUser(u))

	// Get and check
	got, err := us.GetUserByUsername("me")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "administrator", got.Role)
	ok, err := got.PasswordHash.Matches("securepassword")
	require.NoError(t, err)
	assert.True(t, ok)

	// Update
	got.Email = "new@example.com"
	require.NoError(t, us.UpdateUser(got))

	// Get again and check update
	got2, err := us.GetUserByUsername("me")
	require.NoError(t, err)
	assert.Equal(t, "new@example.com", got2.Email)
}

func TestUserStore_GetUserToken(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	us := NewPostgresUserStore(db)

	u := &User{Username: "tok", Email: "tok@ex.com", Role: "employee"}
	require.NoError(t, u.PasswordHash.Set("p"))
	require.NoError(t, us.CreateUser(u))

	const scope = "auth"
	// Valid token
	const validTokenPlain = "token-abc-123"
	validHash := sha256.Sum256([]byte(validTokenPlain))
	_, err := db.Exec(`INSERT INTO tokens (hash, user_id, expiry, scope) VALUES ($1,$2,$3,$4)`,
		validHash[:], u.ID, time.Now().Add(1*time.Hour), scope)
	require.NoError(t, err)

	// Expired token
	const expiredTokenPlain = "expired"
	expHash := sha256.Sum256([]byte(expiredTokenPlain))
	_, err = db.Exec(`INSERT INTO tokens (hash, user_id, expiry, scope) VALUES ($1,$2,$3,$4)`,
		expHash[:], u.ID, time.Now().Add(-1*time.Minute), scope)
	require.NoError(t, err)

	tests := []struct {
		name      string
		token     string
		wantFound bool
		wantErr   bool
	}{
		{name: "valid token", token: validTokenPlain, wantFound: true, wantErr: false},
		{name: "expired token", token: expiredTokenPlain, wantFound: false, wantErr: false},
		{name: "non-existent token", token: "not-real", wantFound: false, wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			foundUser, err := us.GetUserToken(scope, tt.token)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.wantFound {
				require.NotNil(t, foundUser)
				assert.Equal(t, u.ID, foundUser.ID)
			} else {
				assert.Nil(t, foundUser)
			}
		})
	}
}
