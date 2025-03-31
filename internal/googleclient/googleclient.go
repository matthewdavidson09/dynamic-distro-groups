package googleclient

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"golang.org/x/oauth2/google"
	admin "google.golang.org/api/admin/directory/v1"
	"google.golang.org/api/option"
)

func NewDirectoryService(ctx context.Context) (*admin.Service, error) {
	credsPath := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	if credsPath == "" {
		return nil, fmt.Errorf("GOOGLE_APPLICATION_CREDENTIALS env var not set")
	}

	data, err := os.ReadFile(credsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read service account JSON: %w", err)
	}

	impersonateUser := os.Getenv("GOOGLE_IMPERSONATE_USER")
	if impersonateUser == "" {
		return nil, fmt.Errorf("GOOGLE_IMPERSONATE_USER env var not set")
	}

	scopes := []string{
		admin.AdminDirectoryGroupScope,
		admin.AdminDirectoryUserScope,
		"https://www.googleapis.com/auth/apps.groups.settings",
	}

	config, err := google.JWTConfigFromJSON(data, scopes...)
	if err != nil {
		return nil, fmt.Errorf("failed to parse service account credentials: %w", err)
	}
	config.Subject = impersonateUser

	client := config.Client(ctx)

	return admin.NewService(ctx, option.WithHTTPClient(client))
}

func NewImpersonatedHTTPClient(ctx context.Context) (*http.Client, error) {
	credsPath := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	if credsPath == "" {
		return nil, fmt.Errorf("GOOGLE_APPLICATION_CREDENTIALS env var not set")
	}

	data, err := os.ReadFile(credsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read service account JSON: %w", err)
	}

	impersonateUser := os.Getenv("GOOGLE_IMPERSONATE_USER")
	if impersonateUser == "" {
		return nil, fmt.Errorf("GOOGLE_IMPERSONATE_USER env var not set")
	}

	scopes := []string{
		admin.AdminDirectoryGroupScope,
		admin.AdminDirectoryUserScope,
		"https://www.googleapis.com/auth/apps.groups.settings",
	}

	config, err := google.JWTConfigFromJSON(data, scopes...)
	if err != nil {
		return nil, fmt.Errorf("failed to parse service account credentials: %w", err)
	}
	config.Subject = impersonateUser

	return config.Client(ctx), nil
}
