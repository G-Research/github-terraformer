package github

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/google/go-github/v67/github"
	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

func getToken() (string, error) {
	token := os.Getenv("GITHUB_TOKEN")

	if token != "" {
		fmt.Printf("using token from env var")
		return token, nil
	}

	cmd := exec.Command("gh", "auth", "token")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get token: %w", err)
	}

	if token = strings.TrimSpace(string(output)); token != "" {
		return token, nil
	}
	return "", errors.New("retrieved token is empty")
}

func CreateGitHubClient() (*github.Client, *githubv4.Client, error) {
	token, err := getToken()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to retrieve token: %w", err)
	}

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)
	v4client := githubv4.NewClient(tc)
	return client, v4client, nil
}
