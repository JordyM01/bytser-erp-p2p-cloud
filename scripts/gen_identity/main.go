package main

import (
	"context"
	"fmt"
	"os"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"

	"github.com/bytsers/erp-p2p-cloud/internal/node"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	secretName := os.Getenv("P2P_IDENTITY_SECRET_NAME")
	if secretName == "" {
		secretName = "bytsers/p2p-relay/identity"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("loading AWS config: %w", err)
	}

	client := secretsmanager.NewFromConfig(awsCfg)

	id, err := node.GenerateAndSave(ctx, secretName, client)
	if err != nil {
		return fmt.Errorf("generating identity: %w", err)
	}

	fmt.Printf("Identity generated and saved to secret %q\n", secretName)
	fmt.Printf("PeerID: %s\n", id.PeerID)
	return nil
}
