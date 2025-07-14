# Token Fetcher

Token Fetcher fetches access tokens stored by the [Ello Token Rotator](https://github.com/ellogroup/ello-token-rotator-lambda-app).

## Usage

```go
package main

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/ellogroup/ello-golang-token-fetcher/token"
)

func main() {
	ctx := context.Background()
	secretsManagerClient := secretsmanager.New(secretsmanager.Options{})
	secretsManagerKey := "token-key"

	fetcher := token.NewAWSSecretsManagerFetcher(
		secretsManagerClient, // AWS Secrets Manager Client
		secretsManagerKey,    // AWS Secrets Manager key of token
	)

	tok, err := fetcher.Fetch(ctx)
	if err != nil {
		panic(err)
	}
	println(tok)
}
```

### Options

#### Token Expiry Buffer

The token expiry buffer is the duration before an access token expires when the token should be refreshed. Default is 1 minute.

```go
fetcher := token.NewAWSSecretsManagerFetcher(
    secretsManagerClient,                       // AWS Secrets Manager Client
    secretsManagerKey,                          // AWS Secrets Manager key of token
    token.WithTokenExpiryBuffer(5*time.Minute), // Refresh the token 5 minutes before it expires
)
```

### Adapters

#### Interface

The adapter interface allows the access token to be fetched from different sources.

````go
type Adapter interface {
	Fetch(ctx context.Context) (Token, error)
}
````

### Implementations

#### AWS Secrets Manager

The AWS Secrets Manager implementation will fetch the access token from Secrets Manager, which is the default storage 
option. 

```go
fetcher := token.NewAWSSecretsManagerFetcher(
    secretsManagerClient, // AWS Secrets Manager Client
    secretsManagerKey,    // AWS Secrets Manager key of token
)
```

#### Custom

A custom adapter can be provided by implementing the `Adapter` interface.

```go
type custom struct {}

func (c *custom) Fetch(ctx context.Context) (token.Token, error) {
    return token.Token{AccessToken: "token"}, nil
}

customAdapter := &custom{}
fetcher := token.New(customAdapter)
```