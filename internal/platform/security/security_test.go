package security

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testHasher() *Hasher {
	// Minimum OWASP parameters keep the test fast while exercising real code.
	return NewHasher(Argon2Params{MemoryKiB: 19456, Iterations: 2, Parallelism: 1})
}

func TestHashAndVerify(t *testing.T) {
	h := testHasher()
	hash, err := h.Hash("correct horse battery staple")
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(hash, "$argon2id$v=19$m=19456,t=2,p=1$"), "PHC format with parameters: %s", hash)

	require.NoError(t, h.Verify("correct horse battery staple", hash))
	assert.ErrorIs(t, h.Verify("wrong password", hash), ErrHashMismatch)
}

func TestHashUniquePerCall(t *testing.T) {
	h := testHasher()
	h1, err := h.Hash("same password")
	require.NoError(t, err)
	h2, err := h.Hash("same password")
	require.NoError(t, err)
	assert.NotEqual(t, h1, h2, "salts must differ")
}

func TestVerifyRejectsGarbage(t *testing.T) {
	h := testHasher()
	assert.Error(t, h.Verify("pw", "not-a-hash"))
	assert.Error(t, h.Verify("pw", "$bcrypt$whatever"))
}

func TestVerifyUsesStoredParams(t *testing.T) {
	// A hash produced with different (weaker test-only) parameters still
	// verifies: parameter upgrades roll forward without breaking old hashes.
	old := NewHasher(Argon2Params{MemoryKiB: 19456, Iterations: 2, Parallelism: 1})
	hash, err := old.Hash("pw12345678")
	require.NoError(t, err)
	upgraded := NewHasher(Argon2Params{MemoryKiB: 65536, Iterations: 3, Parallelism: 2})
	assert.NoError(t, upgraded.Verify("pw12345678", hash))
}

func TestTokenSignVerify(t *testing.T) {
	tm := NewTokenManager("test-secret-key-with-enough-bytes!!", 15*time.Minute)
	userID, sessionID := uuid.New(), uuid.New()
	token, err := tm.Sign(userID, sessionID, []string{"customer", "admin"}, time.Now())
	require.NoError(t, err)

	gotUser, gotSession, roles, err := tm.Verify(token)
	require.NoError(t, err)
	assert.Equal(t, userID, gotUser)
	assert.Equal(t, sessionID, gotSession)
	assert.Equal(t, []string{"customer", "admin"}, roles)
}

func TestTokenExpired(t *testing.T) {
	tm := NewTokenManager("test-secret-key-with-enough-bytes!!", time.Minute)
	token, err := tm.Sign(uuid.New(), uuid.New(), nil, time.Now().Add(-time.Hour))
	require.NoError(t, err)
	_, _, _, err = tm.Verify(token)
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestTokenWrongSecret(t *testing.T) {
	tm := NewTokenManager("test-secret-key-with-enough-bytes!!", time.Minute)
	other := NewTokenManager("a-completely-different-secret-key!!!", time.Minute)
	token, err := tm.Sign(uuid.New(), uuid.New(), nil, time.Now())
	require.NoError(t, err)
	_, _, _, err = other.Verify(token)
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestTokenAlgorithmPinned(t *testing.T) {
	tm := NewTokenManager("test-secret-key-with-enough-bytes!!", time.Minute)
	// alg=none style forgery: header {"alg":"none","typ":"JWT"}.
	forged := "eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0.eyJzdWIiOiJ4In0."
	_, _, _, err := tm.Verify(forged)
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestRandomToken(t *testing.T) {
	a, err := RandomToken()
	require.NoError(t, err)
	b, err := RandomToken()
	require.NoError(t, err)
	assert.NotEqual(t, a, b)
	assert.GreaterOrEqual(t, len(a), 43) // 32 bytes base64url
}

func TestRandomDigits(t *testing.T) {
	code, err := RandomDigits(6)
	require.NoError(t, err)
	assert.Len(t, code, 6)
	for _, c := range code {
		assert.True(t, c >= '0' && c <= '9')
	}
}

func TestHashTokenDeterministic(t *testing.T) {
	assert.Equal(t, HashToken("abc"), HashToken("abc"))
	assert.NotEqual(t, HashToken("abc"), HashToken("abd"))
}

func TestRandomDigitsAreUniform(t *testing.T) {
	// A rough check only: 60k digits should each land near 6000 (σ ≈ 73,
	// so ±300 is about 4σ). The modulo bias this guards against is smaller
	// than that; the redraw in RandomDigits is what removes it.
	var counts [10]int
	for range 10000 {
		code, err := RandomDigits(6)
		require.NoError(t, err)
		require.Len(t, code, 6)
		for _, c := range code {
			counts[c-'0']++
		}
	}
	for d, n := range counts {
		assert.InDelta(t, 6000, n, 300, "digit %d", d)
	}
}

func TestCodeHasher(t *testing.T) {
	h := NewCodeHasher("server-secret-at-least-32-bytes-long!!")
	u1, u2 := uuid.New(), uuid.New()
	sum := h.Hash(u1, "password_reset", "123456")

	assert.Equal(t, sum, h.Hash(u1, "password_reset", "123456"), "deterministic")
	assert.Len(t, sum, 32)
	assert.NotEqual(t, sum, h.Hash(u1, "password_reset", "123457"), "code")
	assert.NotEqual(t, sum, h.Hash(u2, "password_reset", "123456"), "user")
	assert.NotEqual(t, sum, h.Hash(u1, "email_verify", "123456"), "purpose")
	assert.NotEqual(t, sum, NewCodeHasher("another-secret-at-least-32-bytes!!").Hash(u1, "password_reset", "123456"), "key")
	assert.NotEqual(t, sum, HashToken("123456"), "not a plain digest")
}
