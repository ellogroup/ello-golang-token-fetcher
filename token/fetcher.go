package token

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/ellogroup/ello-golang-clock/clock"
	"time"
)

type Token struct {
	AccessToken  string    `json:"access_token"`
	TokenType    string    `json:"token_type,omitempty"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	Expiry       time.Time `json:"expiry,omitempty"`
	CreatedAt    time.Time `json:"created_at,omitempty"`
}

type Fetcher struct {
	config  config
	clock   clock.Clock
	adapter Adapter
	token   Token
}

type config struct {
	tokenExpiryBuffer time.Duration
}

var defaultConfig = config{
	tokenExpiryBuffer: time.Minute,
}

type Option func(*config)

func WithTokenExpiryBuffer(buffer time.Duration) Option {
	return func(c *config) { c.tokenExpiryBuffer = buffer }
}

func New(adapter Adapter, opts ...Option) *Fetcher {
	c := defaultConfig
	for _, opt := range opts {
		opt(&c)
	}
	return &Fetcher{
		config:  c,
		clock:   clock.NewSystem(),
		adapter: adapter,
	}
}

func NewAWSSecretsManagerFetcher(smClient *secretsmanager.Client, smKey string, opts ...Option) *Fetcher {
	return New(awsSecretsManagerAdapter{
		client: smClient,
		key:    smKey,
	},
		opts...,
	)
}

func (f *Fetcher) Fetch(ctx context.Context) (Token, error) {
	if f.refreshRequired() {
		return f.refresh(ctx)
	}
	return f.token, nil
}

func (f *Fetcher) refreshRequired() bool {
	return f.token.AccessToken == "" || (!f.token.Expiry.IsZero() && f.token.Expiry.Before(f.clock.Now().Add(f.config.tokenExpiryBuffer)))
}

func (f *Fetcher) refresh(ctx context.Context) (Token, error) {
	t, err := f.adapter.Fetch(ctx)
	if err != nil {
		return Token{}, err
	}

	f.token = t
	return t, nil
}

type Adapter interface {
	Fetch(ctx context.Context) (Token, error)
}

type awsSecretsManagerClient interface {
	GetSecretValue(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error)
}
type awsSecretsManagerAdapter struct {
	client awsSecretsManagerClient
	key    string
}

func (a awsSecretsManagerAdapter) Fetch(ctx context.Context) (Token, error) {
	out, err := a.client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(a.key),
	})
	if err != nil {
		return Token{}, fmt.Errorf("unable to fetch token from secrets manager: %w", err)
	}

	var t Token
	if err := json.Unmarshal([]byte(*out.SecretString), &t); err != nil {
		return Token{}, fmt.Errorf("unable to parse token from secrets manager: %w", err)
	}

	return t, nil
}
