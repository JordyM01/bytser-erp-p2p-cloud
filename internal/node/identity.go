package node

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	smtypes "github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
)

// Identity holds the Ed25519 keypair and derived PeerID for this node.
type Identity struct {
	PrivKey crypto.PrivKey
	PubKey  crypto.PubKey
	PeerID  peer.ID
}

// SecretsClient is the subset of the AWS Secrets Manager API used for identity storage.
type SecretsClient interface {
	GetSecretValue(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error)
	CreateSecret(ctx context.Context, params *secretsmanager.CreateSecretInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.CreateSecretOutput, error)
}

func newIdentityFromPrivKey(privKey crypto.PrivKey) (*Identity, error) {
	pubKey := privKey.GetPublic()
	pid, err := peer.IDFromPublicKey(pubKey)
	if err != nil {
		return nil, fmt.Errorf("deriving peer ID: %w", err)
	}
	return &Identity{
		PrivKey: privKey,
		PubKey:  pubKey,
		PeerID:  pid,
	}, nil
}

// LoadOrGenerateLocal loads an Ed25519 identity from keyFilePath.
// If the file does not exist, it generates a new keypair and saves it.
func LoadOrGenerateLocal(keyFilePath string) (*Identity, error) {
	data, err := os.ReadFile(keyFilePath)
	if err == nil {
		privKey, err := crypto.UnmarshalPrivateKey(data)
		if err != nil {
			return nil, fmt.Errorf("unmarshaling private key from %s: %w", keyFilePath, err)
		}
		return newIdentityFromPrivKey(privKey)
	}

	if !os.IsNotExist(err) {
		return nil, fmt.Errorf("reading key file %s: %w", keyFilePath, err)
	}

	privKey, _, err := crypto.GenerateEd25519Key(nil)
	if err != nil {
		return nil, fmt.Errorf("generating Ed25519 key: %w", err)
	}

	raw, err := crypto.MarshalPrivateKey(privKey)
	if err != nil {
		return nil, fmt.Errorf("marshaling private key: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(keyFilePath), 0o700); err != nil {
		return nil, fmt.Errorf("creating directory for key file: %w", err)
	}

	if err := os.WriteFile(keyFilePath, raw, 0o600); err != nil {
		return nil, fmt.Errorf("writing key file %s: %w", keyFilePath, err)
	}

	return newIdentityFromPrivKey(privKey)
}

type secretJSON struct {
	PrivateKeyB64 string `json:"private_key_b64"`
}

// LoadFromSecretsManager loads an Ed25519 identity from AWS Secrets Manager.
// The secret must contain JSON: {"private_key_b64": "<base64-encoded protobuf>"}.
func LoadFromSecretsManager(ctx context.Context, secretName string, client SecretsClient) (*Identity, error) {
	out, err := client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: &secretName,
	})
	if err != nil {
		var notFound *smtypes.ResourceNotFoundException
		if errors.As(err, &notFound) {
			return nil, fmt.Errorf("secret %q not found: %w", secretName, err)
		}
		return nil, fmt.Errorf("getting secret %q: %w", secretName, err)
	}

	var sj secretJSON
	if err := json.Unmarshal([]byte(*out.SecretString), &sj); err != nil {
		return nil, fmt.Errorf("parsing secret JSON: %w", err)
	}

	raw, err := base64.StdEncoding.DecodeString(sj.PrivateKeyB64)
	if err != nil {
		return nil, fmt.Errorf("decoding base64 private key: %w", err)
	}

	privKey, err := crypto.UnmarshalPrivateKey(raw)
	if err != nil {
		return nil, fmt.Errorf("unmarshaling private key: %w", err)
	}

	return newIdentityFromPrivKey(privKey)
}

// GenerateAndSave generates a new Ed25519 keypair and stores it in AWS Secrets Manager.
func GenerateAndSave(ctx context.Context, secretName string, client SecretsClient) (*Identity, error) {
	privKey, _, err := crypto.GenerateEd25519Key(nil)
	if err != nil {
		return nil, fmt.Errorf("generating Ed25519 key: %w", err)
	}

	raw, err := crypto.MarshalPrivateKey(privKey)
	if err != nil {
		return nil, fmt.Errorf("marshaling private key: %w", err)
	}

	sj := secretJSON{
		PrivateKeyB64: base64.StdEncoding.EncodeToString(raw),
	}
	secretBytes, err := json.Marshal(sj)
	if err != nil {
		return nil, fmt.Errorf("marshaling secret JSON: %w", err)
	}

	secretString := string(secretBytes)
	_, err = client.CreateSecret(ctx, &secretsmanager.CreateSecretInput{
		Name:         &secretName,
		SecretString: &secretString,
	})
	if err != nil {
		var exists *smtypes.ResourceExistsException
		if errors.As(err, &exists) {
			return nil, fmt.Errorf("secret %q already exists: %w", secretName, err)
		}
		return nil, fmt.Errorf("creating secret %q: %w", secretName, err)
	}

	return newIdentityFromPrivKey(privKey)
}
