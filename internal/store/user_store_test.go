package store

import (
	"crypto/sha256"
	"database/sql"
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

func TestUserStore(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T, db *sql.DB)
	}{
		{
			name: "create_get_update",
			run: func(t *testing.T, db *sql.DB) {
				us := NewPostgresUserStore(db)

				u := &User{Username: "me", Email: "me@example.com"}
				require.NoError(t, u.PasswordHash.Set("securepassword"))
				require.NoError(t, us.CreateUser(u))
				assert.NotZero(t, u.ID)

				got, err := us.GetUserByUsername("me")
				require.NoError(t, err)
				require.NotNil(t, got)

				ok, err := got.PasswordHash.Matches("securepassword")
				require.NoError(t, err)
				assert.True(t, ok)

				got.Email = "new@example.com"
				require.NoError(t, us.UpdateUser(got))

				got2, err := us.GetUserByUsername("me")
				require.NoError(t, err)
				assert.Equal(t, "new@example.com", got2.Email)
			},
		},
		{
			name: "duplicate_username",
			run: func(t *testing.T, db *sql.DB) {
				us := NewPostgresUserStore(db)

				u1 := &User{Username: "dup", Email: "a@ex.com"}
				require.NoError(t, u1.PasswordHash.Set("x"))
				require.NoError(t, us.CreateUser(u1))

				u2 := &User{Username: "dup", Email: "b@ex.com"}
				require.NoError(t, u2.PasswordHash.Set("y"))
				err := us.CreateUser(u2)
				require.Error(t, err)
			},
		},
		{
			name: "token_valid_and_expired",
			run: func(t *testing.T, db *sql.DB) {
				us := NewPostgresUserStore(db)

				u := &User{Username: "tok", Email: "tok@ex.com"}
				require.NoError(t, u.PasswordHash.Set("p"))
				require.NoError(t, us.CreateUser(u))

				const scope = "auth"
				const tokenPlain = "token-abc-123"
				hash := sha256.Sum256([]byte(tokenPlain))
				_, err := db.Exec(`INSERT INTO tokens (hash, user_id, expiry, scope) VALUES ($1,$2,$3,$4)`,
					hash[:], u.ID, time.Now().Add(1*time.Hour), scope)
				require.NoError(t, err)

				got, err := us.GetUserToken(scope, tokenPlain)
				require.NoError(t, err)
				require.NotNil(t, got)
				assert.Equal(t, u.ID, got.ID)

				expHash := sha256.Sum256([]byte("expired"))
				_, err = db.Exec(`INSERT INTO tokens (hash, user_id, expiry, scope) VALUES ($1,$2,$3,$4)`,
					expHash[:], u.ID, time.Now().Add(-1*time.Minute), scope)
				require.NoError(t, err)

				nilUser, err := us.GetUserToken(scope, "expired")
				require.NoError(t, err)
				assert.Nil(t, nilUser)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupTestDB(t)
			defer db.Close()
			tt.run(t, db)
		})
	}
}
