package node

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	smtypes "github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockSecretsClient struct {
	getSecretFunc    func(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error)
	createSecretFunc func(ctx context.Context, params *secretsmanager.CreateSecretInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.CreateSecretOutput, error)
}

func (m *mockSecretsClient) GetSecretValue(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
	return m.getSecretFunc(ctx, params, optFns...)
}

func (m *mockSecretsClient) CreateSecret(ctx context.Context, params *secretsmanager.CreateSecretInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.CreateSecretOutput, error) {
	return m.createSecretFunc(ctx, params, optFns...)
}

// generateTestKeyJSON creates a real Ed25519 keypair and returns the JSON secret string and expected PeerID.
func generateTestKeyJSON(t *testing.T) (string, peer.ID) {
	t.Helper()
	privKey, _, err := crypto.GenerateEd25519Key(nil)
	require.NoError(t, err)

	raw, err := crypto.MarshalPrivateKey(privKey)
	require.NoError(t, err)

	pid, err := peer.IDFromPublicKey(privKey.GetPublic())
	require.NoError(t, err)

	sj := secretJSON{PrivateKeyB64: base64.StdEncoding.EncodeToString(raw)}
	data, err := json.Marshal(sj)
	require.NoError(t, err)

	return string(data), pid
}

func TestLoadOrGenerateLocal_GeneratesWhenMissing(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "identity.key")

	id, err := LoadOrGenerateLocal(keyPath)
	require.NoError(t, err)
	assert.NotEmpty(t, id.PeerID)
	assert.NotNil(t, id.PrivKey)
	assert.NotNil(t, id.PubKey)

	_, err = os.Stat(keyPath)
	assert.NoError(t, err, "key file should have been created")
}

func TestLoadOrGenerateLocal_LoadsExisting(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "identity.key")

	id1, err := LoadOrGenerateLocal(keyPath)
	require.NoError(t, err)

	id2, err := LoadOrGenerateLocal(keyPath)
	require.NoError(t, err)

	assert.Equal(t, id1.PeerID, id2.PeerID, "reloaded identity should have the same PeerID")
}

func TestLoadOrGenerateLocal_DeterministicPeerID(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "identity.key")

	id1, err := LoadOrGenerateLocal(keyPath)
	require.NoError(t, err)

	// Read key file directly and derive PeerID
	data, err := os.ReadFile(keyPath)
	require.NoError(t, err)

	privKey, err := crypto.UnmarshalPrivateKey(data)
	require.NoError(t, err)

	pid, err := peer.IDFromPublicKey(privKey.GetPublic())
	require.NoError(t, err)

	assert.Equal(t, id1.PeerID, pid, "PeerID derived manually should match")
}

func TestLoadFromSecretsManager_Success(t *testing.T) {
	secretStr, expectedPID := generateTestKeyJSON(t)

	mock := &mockSecretsClient{
		getSecretFunc: func(_ context.Context, _ *secretsmanager.GetSecretValueInput, _ ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
			return &secretsmanager.GetSecretValueOutput{SecretString: &secretStr}, nil
		},
	}

	id, err := LoadFromSecretsManager(context.Background(), "test-secret", mock)
	require.NoError(t, err)
	assert.Equal(t, expectedPID, id.PeerID)
}

func TestLoadFromSecretsManager_SecretNotFound(t *testing.T) {
	mock := &mockSecretsClient{
		getSecretFunc: func(_ context.Context, _ *secretsmanager.GetSecretValueInput, _ ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
			return nil, &smtypes.ResourceNotFoundException{Message: stringPtr("not found")}
		},
	}

	_, err := LoadFromSecretsManager(context.Background(), "missing-secret", mock)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestLoadFromSecretsManager_InvalidJSON(t *testing.T) {
	garbage := "not-json-at-all"
	mock := &mockSecretsClient{
		getSecretFunc: func(_ context.Context, _ *secretsmanager.GetSecretValueInput, _ ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
			return &secretsmanager.GetSecretValueOutput{SecretString: &garbage}, nil
		},
	}

	_, err := LoadFromSecretsManager(context.Background(), "test-secret", mock)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parsing secret JSON")
}

func TestLoadFromSecretsManager_InvalidBase64(t *testing.T) {
	badB64 := `{"private_key_b64": "!!!invalid-base64!!!"}`
	mock := &mockSecretsClient{
		getSecretFunc: func(_ context.Context, _ *secretsmanager.GetSecretValueInput, _ ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
			return &secretsmanager.GetSecretValueOutput{SecretString: &badB64}, nil
		},
	}

	_, err := LoadFromSecretsManager(context.Background(), "test-secret", mock)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decoding base64")
}

func TestGenerateAndSave_Success(t *testing.T) {
	var captured *secretsmanager.CreateSecretInput
	mock := &mockSecretsClient{
		createSecretFunc: func(_ context.Context, params *secretsmanager.CreateSecretInput, _ ...func(*secretsmanager.Options)) (*secretsmanager.CreateSecretOutput, error) {
			captured = params
			return &secretsmanager.CreateSecretOutput{}, nil
		},
	}

	id, err := GenerateAndSave(context.Background(), "test-secret", mock)
	require.NoError(t, err)
	assert.NotEmpty(t, id.PeerID)

	// Verify the captured secret is valid JSON with base64 key
	require.NotNil(t, captured)
	assert.Equal(t, "test-secret", *captured.Name)

	var sj secretJSON
	err = json.Unmarshal([]byte(*captured.SecretString), &sj)
	require.NoError(t, err)
	assert.NotEmpty(t, sj.PrivateKeyB64)

	// Verify we can decode the stored key back
	raw, err := base64.StdEncoding.DecodeString(sj.PrivateKeyB64)
	require.NoError(t, err)
	privKey, err := crypto.UnmarshalPrivateKey(raw)
	require.NoError(t, err)
	pid, err := peer.IDFromPublicKey(privKey.GetPublic())
	require.NoError(t, err)
	assert.Equal(t, id.PeerID, pid, "stored key should match returned identity")
}

func TestGenerateAndSave_AlreadyExists(t *testing.T) {
	mock := &mockSecretsClient{
		createSecretFunc: func(_ context.Context, _ *secretsmanager.CreateSecretInput, _ ...func(*secretsmanager.Options)) (*secretsmanager.CreateSecretOutput, error) {
			return nil, &smtypes.ResourceExistsException{Message: stringPtr("already exists")}
		},
	}

	_, err := GenerateAndSave(context.Background(), "existing-secret", mock)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func stringPtr(s string) *string { return &s }

func TestLoadOrGenerateLocal_CreatesSubdirectories(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "sub", "dir", "identity.key")

	id, err := LoadOrGenerateLocal(keyPath)
	require.NoError(t, err)
	assert.NotEmpty(t, id.PeerID)

	info, err := os.Stat(keyPath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm(), "key file should have 0600 permissions")
}

func TestLoadOrGenerateLocal_RejectsCorruptFile(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "identity.key")

	err := os.WriteFile(keyPath, []byte("corrupt-data"), 0o600)
	require.NoError(t, err)

	_, err = LoadOrGenerateLocal(keyPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshaling private key")
}

func TestLoadFromSecretsManager_InvalidKeyBytes(t *testing.T) {
	// Valid JSON, valid base64, but the decoded bytes are not a valid protobuf key
	badKey := base64.StdEncoding.EncodeToString([]byte("not-a-real-key"))
	secretStr := fmt.Sprintf(`{"private_key_b64": %q}`, badKey)
	mock := &mockSecretsClient{
		getSecretFunc: func(_ context.Context, _ *secretsmanager.GetSecretValueInput, _ ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
			return &secretsmanager.GetSecretValueOutput{SecretString: &secretStr}, nil
		},
	}

	_, err := LoadFromSecretsManager(context.Background(), "test-secret", mock)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshaling private key")
}
