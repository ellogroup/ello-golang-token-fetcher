package token

import (
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/ellogroup/ello-golang-clock/clock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
	"time"
)

type mockAdapter struct {
	mock.Mock
}

func (m *mockAdapter) Fetch(ctx context.Context) (Token, error) {
	args := m.Called(ctx)
	return args.Get(0).(Token), args.Error(1)
}

func TestFetcher_Fetch(t *testing.T) {
	now := time.Date(2030, 1, 2, 0, 0, 0, 0, time.UTC)
	tok := Token{AccessToken: "token-123"}

	type fields struct {
		config config
		token  Token
	}
	type args struct {
		ctx context.Context
	}
	type mockOpts struct {
		adapter func(m *mockAdapter)
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		mockOpts mockOpts
		want     Token
		wantErr  assert.ErrorAssertionFunc
	}{
		{
			name:    "valid token, returns token",
			fields:  fields{config: defaultConfig, token: tok},
			args:    args{context.Background()},
			want:    tok,
			wantErr: assert.NoError,
		},
		{
			name:   "missing token, returns new token",
			fields: fields{config: defaultConfig},
			args:   args{context.Background()},
			mockOpts: mockOpts{func(m *mockAdapter) {
				m.On("Fetch", mock.Anything).Return(tok, nil).Once()
			}},
			want:    tok,
			wantErr: assert.NoError,
		},
		{
			name:   "missing token, adapter returns error, returns error",
			fields: fields{config: defaultConfig},
			args:   args{context.Background()},
			mockOpts: mockOpts{func(m *mockAdapter) {
				m.On("Fetch", mock.Anything).Return(Token{}, errors.New("error")).Once()
			}},
			wantErr: assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mAdapter := new(mockAdapter)
			if tt.mockOpts.adapter != nil {
				tt.mockOpts.adapter(mAdapter)
			}

			f := &Fetcher{
				config:  tt.fields.config,
				clock:   clock.NewFixed(now),
				adapter: mAdapter,
				token:   tt.fields.token,
			}
			got, err := f.Fetch(tt.args.ctx)
			if !tt.wantErr(t, err, fmt.Sprintf("Fetch(%v)", tt.args.ctx)) {
				return
			}
			assert.Equalf(t, tt.want, got, "Fetch(%v)", tt.args.ctx)
		})
	}
}

func TestFetcher_refreshRequired(t *testing.T) {
	now := time.Date(2030, 1, 2, 0, 0, 0, 0, time.UTC)
	past := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	future := time.Date(2030, 1, 2, 1, 0, 0, 0, time.UTC)

	type fields struct {
		config config
		clock  clock.Clock
		token  Token
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "empty token, returns true",
			fields: fields{
				config: config{tokenExpiryBuffer: time.Minute},
				clock:  clock.NewFixed(now),
				token:  Token{},
			},
			want: true,
		},
		{
			name: "token exists, expiry set in the past, returns true",
			fields: fields{
				config: config{tokenExpiryBuffer: time.Minute},
				clock:  clock.NewFixed(now),
				token:  Token{AccessToken: "token-123", Expiry: past},
			},
			want: true,
		},
		{
			name: "token exists, expiry set as now with expiry buffer set, returns true",
			fields: fields{
				config: config{tokenExpiryBuffer: time.Minute},
				clock:  clock.NewFixed(now),
				token:  Token{AccessToken: "token-123", Expiry: now},
			},
			want: true,
		},
		{
			name: "token exists, expiry set as now with no expiry buffer set, returns false",
			fields: fields{
				config: config{tokenExpiryBuffer: 0},
				clock:  clock.NewFixed(now),
				token:  Token{AccessToken: "token-123", Expiry: now},
			},
			want: false,
		},
		{
			name: "token exists, expiry set in the future, returns false",
			fields: fields{
				config: config{tokenExpiryBuffer: time.Minute},
				clock:  clock.NewFixed(now),
				token:  Token{AccessToken: "token-123", Expiry: future},
			},
			want: false,
		},
		{
			name: "token exists, no expiry set, returns false",
			fields: fields{
				config: config{tokenExpiryBuffer: time.Minute},
				clock:  clock.NewFixed(now),
				token:  Token{AccessToken: "token-123"},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &Fetcher{
				config: tt.fields.config,
				clock:  clock.NewFixed(now),
				token:  tt.fields.token,
			}
			assert.Equalf(t, tt.want, f.refreshRequired(), "refreshRequired()")
		})
	}
}

func TestFetcher_refresh(t *testing.T) {
	tok := Token{AccessToken: "token-123"}

	type fields struct {
		token Token
	}
	type args struct {
		ctx context.Context
	}
	type mockOpts struct {
		adapter func(m *mockAdapter)
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		mockOpts mockOpts
		want     Token
		wantErr  assert.ErrorAssertionFunc
	}{
		{
			name: "adapter returns token, returns token",
			args: args{context.Background()},
			mockOpts: mockOpts{func(m *mockAdapter) {
				m.On("Fetch", mock.Anything).Return(tok, nil).Once()
			}},
			want:    tok,
			wantErr: assert.NoError,
		},
		{
			name:   "adapter returns token, cached token already exists, returns token and cached token overwritten",
			fields: fields{token: Token{AccessToken: "another-token-123"}},
			args:   args{context.Background()},
			mockOpts: mockOpts{func(m *mockAdapter) {
				m.On("Fetch", mock.Anything).Return(tok, nil).Once()
			}},
			want:    tok,
			wantErr: assert.NoError,
		},
		{
			name: "adapter returns error, returns error",
			args: args{context.Background()},
			mockOpts: mockOpts{func(m *mockAdapter) {
				m.On("Fetch", mock.Anything).Return(Token{}, errors.New("error")).Once()
			}},
			wantErr: assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mAdapter := new(mockAdapter)
			if tt.mockOpts.adapter != nil {
				tt.mockOpts.adapter(mAdapter)
			}

			f := &Fetcher{
				adapter: mAdapter,
				token:   tt.fields.token,
			}
			got, err := f.refresh(tt.args.ctx)
			if !tt.wantErr(t, err, fmt.Sprintf("refresh(%v)", tt.args.ctx)) {
				return
			}
			assert.Equalf(t, tt.want, got, "refresh(%v)", tt.args.ctx)
			assert.Equalf(t, tt.want, f.token, "refresh(%v)", tt.args.ctx)
		})
	}
}

type mockAWSSecretsManagerClient struct {
	mock.Mock
}

func (m *mockAWSSecretsManagerClient) GetSecretValue(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
	args := m.Called(ctx, params, optFns)
	return args.Get(0).(*secretsmanager.GetSecretValueOutput), args.Error(1)
}

func Test_awsSecretsManagerAdapter_Fetch(t *testing.T) {
	type fields struct {
		key string
	}
	type args struct {
		ctx context.Context
	}
	type mockOpts struct {
		client func(m *mockAWSSecretsManagerClient)
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		mockOpts mockOpts
		want     Token
		wantErr  assert.ErrorAssertionFunc
	}{
		{
			name:   "secrets manager returns valid secret, returns token",
			fields: fields{key: "secret-key"},
			args:   args{ctx: context.Background()},
			mockOpts: mockOpts{func(m *mockAWSSecretsManagerClient) {
				m.On("GetSecretValue", mock.Anything, mock.MatchedBy(func(in *secretsmanager.GetSecretValueInput) bool {
					return *in.SecretId == "secret-key"
				}), mock.Anything).Return(&secretsmanager.GetSecretValueOutput{
					SecretString: aws.String(`{"access_token":"token-123","token_type":"bearer","refresh_token":"refresh-123","expiry":"2030-01-02T00:00:00Z","created_at":"2025-01-02T00:00:00Z"}`),
				}, nil).Once()
			}},
			want: Token{
				AccessToken:  "token-123",
				TokenType:    "bearer",
				RefreshToken: "refresh-123",
				Expiry:       time.Date(2030, 1, 2, 0, 0, 0, 0, time.UTC),
				CreatedAt:    time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC),
			},
			wantErr: assert.NoError,
		},
		{
			name:   "secrets manager returns invalid secret, returns error",
			fields: fields{key: "secret-key"},
			args:   args{ctx: context.Background()},
			mockOpts: mockOpts{func(m *mockAWSSecretsManagerClient) {
				m.On("GetSecretValue", mock.Anything, mock.Anything, mock.Anything).Return(&secretsmanager.GetSecretValueOutput{
					SecretString: aws.String(`{invalid-json]`),
				}, nil).Once()
			}},
			wantErr: assert.Error,
		},
		{
			name:   "secrets manager returns error, returns error",
			fields: fields{key: "secret-key"},
			args:   args{ctx: context.Background()},
			mockOpts: mockOpts{func(m *mockAWSSecretsManagerClient) {
				m.On("GetSecretValue", mock.Anything, mock.Anything, mock.Anything).Return(&secretsmanager.GetSecretValueOutput{}, errors.New("secrets manager error")).Once()
			}},
			wantErr: assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mClient := new(mockAWSSecretsManagerClient)
			if tt.mockOpts.client != nil {
				tt.mockOpts.client(mClient)
			}

			a := awsSecretsManagerAdapter{
				client: mClient,
				key:    tt.fields.key,
			}
			got, err := a.Fetch(tt.args.ctx)
			if !tt.wantErr(t, err, fmt.Sprintf("Fetch(%v)", tt.args.ctx)) {
				return
			}
			assert.Equalf(t, tt.want, got, "Fetch(%v)", tt.args.ctx)
		})
	}
}
