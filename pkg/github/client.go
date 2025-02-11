package github

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/google/go-github/v60/github"
)

// Client defines the interface for GitHub operations
type Client interface {
	// Secret operations
	CreateOrUpdateSecret(ctx context.Context, owner, repo, name string, value []byte) error
	DeleteSecret(ctx context.Context, owner, repo, name string) error

	// Variable operations
	CreateOrUpdateVariable(ctx context.Context, owner, repo, name, value string) error
	DeleteVariable(ctx context.Context, owner, repo, name string) error
}

// Config holds the GitHub App configuration
type Config struct {
	AppID          int64
	InstallationID int64
	PrivateKey     []byte
}

type client struct {
	ghClient *github.Client
	config   Config
}

// NewClient creates a new GitHub client using GitHub App authentication
func NewClient(config Config) (Client, error) {
	// Create GitHub App transport
	itr, err := ghinstallation.New(
		http.DefaultTransport,
		config.AppID,
		config.InstallationID,
		config.PrivateKey,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub App transport: %w", err)
	}

	// Create GitHub client with retry and rate limit handling
	httpClient := &http.Client{
		Transport: &retryTransport{
			base: itr,
		},
	}

	return &client{
		ghClient: github.NewClient(httpClient),
		config:   config,
	}, nil
}

// CreateOrUpdateSecret creates or updates a GitHub Actions secret
func (c *client) CreateOrUpdateSecret(ctx context.Context, owner, repo, name string, value []byte) error {
	// Get public key for secret encryption
	key, _, err := c.ghClient.Actions.GetRepoPublicKey(ctx, owner, repo)
	if err != nil {
		return fmt.Errorf("failed to get repository public key: %w", err)
	}

	// Encrypt secret value using sodium library
	encryptedBytes, err := encryptSecretWithPublicKey(value, key.GetKey())
	if err != nil {
		return fmt.Errorf("failed to encrypt secret: %w", err)
	}

	// Create or update secret
	secret := &github.EncryptedSecret{
		Name:           name,
		KeyID:          key.GetKeyID(),
		EncryptedValue: encryptedBytes,
	}
	_, err = c.ghClient.Actions.CreateOrUpdateRepoSecret(ctx, owner, repo, secret)
	if err != nil {
		return fmt.Errorf("failed to create/update secret: %w", err)
	}

	return nil
}

// DeleteSecret deletes a GitHub Actions secret
func (c *client) DeleteSecret(ctx context.Context, owner, repo, name string) error {
	_, err := c.ghClient.Actions.DeleteRepoSecret(ctx, owner, repo, name)
	if err != nil {
		return fmt.Errorf("failed to delete secret: %w", err)
	}
	return nil
}

// CreateOrUpdateVariable creates or updates a GitHub Actions variable
func (c *client) CreateOrUpdateVariable(ctx context.Context, owner, repo, name, value string) error {
	// Use the raw request method since the GitHub API client doesn't have variable methods yet
	url := fmt.Sprintf("repos/%v/%v/actions/variables/%v", owner, repo, name)
	payload := struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	}{
		Name:  name,
		Value: value,
	}

	req, err := c.ghClient.NewRequest("PATCH", url, payload)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.ghClient.Do(ctx, req, nil)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			// Variable doesn't exist, create it
			req, err = c.ghClient.NewRequest("POST", fmt.Sprintf("repos/%v/%v/actions/variables", owner, repo), payload)
			if err != nil {
				return fmt.Errorf("failed to create request: %w", err)
			}
			_, err = c.ghClient.Do(ctx, req, nil)
			if err != nil {
				return fmt.Errorf("failed to create variable: %w", err)
			}
		} else {
			return fmt.Errorf("failed to update variable: %w", err)
		}
	}

	return nil
}

// DeleteVariable deletes a GitHub Actions variable
func (c *client) DeleteVariable(ctx context.Context, owner, repo, name string) error {
	_, err := c.ghClient.Actions.DeleteRepoVariable(ctx, owner, repo, name)
	if err != nil {
		return fmt.Errorf("failed to delete variable: %w", err)
	}
	return nil
}

// retryTransport implements a custom transport with retry logic and rate limit handling
type retryTransport struct {
	base http.RoundTripper
}

func (t *retryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error
	maxRetries := 3
	backoff := time.Second

	for i := 0; i < maxRetries; i++ {
		resp, err = t.base.RoundTrip(req)
		if err != nil {
			return nil, err
		}

		// Check if we hit rate limiting
		if resp.StatusCode == http.StatusForbidden {
			rateLimitReset := resp.Header.Get("X-RateLimit-Reset")
			if rateLimitReset != "" {
				resetTime, parseErr := strconv.ParseInt(rateLimitReset, 10, 64)
				if parseErr == nil {
					waitDuration := time.Until(time.Unix(resetTime, 0))
					if waitDuration > 0 {
						time.Sleep(waitDuration)
						continue
					}
				}
			}
		}

		// If we get a server error, retry with exponential backoff
		if resp.StatusCode >= 500 && i < maxRetries-1 {
			time.Sleep(backoff)
			backoff *= 2
			continue
		}

		break
	}

	return resp, err
}
