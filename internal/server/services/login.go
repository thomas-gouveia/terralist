package services

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"

	"terralist/internal/server/models/oauth"
	"terralist/pkg/auth"
	"terralist/pkg/auth/jwt"
)

var (
	tokenExpirationInSeconds = 24 * 60 * 60
)

// LoginService describes a service that holds the business logic for authentication.
type LoginService interface {
	// Authorize initiates the OAUTH 2.0 process, computing the provider authorize URL.
	Authorize(state oauth.Payload) (string, oauth.Error)

	// UnpackCode uses the code received from the OAUTH 2.0 callback and generates
	// the code components.
	UnpackCode(code string, r *oauth.Request) (*oauth.CodeComponents, oauth.Error)

	// Redirect converts the code components into a redirect URL.
	Redirect(cc *oauth.CodeComponents, r *oauth.Request) (string, oauth.Error)

	// ValidateToken is the method called on the third step from the OAUTH 2.0 protocol.
	// It verifies the code components and generates the authorization token.
	ValidateToken(components *oauth.CodeComponents, verifier string) (*oauth.TokenResponse, oauth.Error)
}

type DefaultLoginService struct {
	Provider auth.Provider
	JWT      jwt.JWT

	EncryptSalt     string
	CodeExchangeKey string
}

func (s *DefaultLoginService) Authorize(state oauth.Payload) (string, oauth.Error) {
	return s.Provider.GetAuthorizeUrl(string(state)), nil
}

func (s *DefaultLoginService) UnpackCode(code string, r *oauth.Request) (*oauth.CodeComponents, oauth.Error) {
	var userDetails auth.User
	if err := s.Provider.GetUserDetails(code, &userDetails); err != nil {
		return nil, oauth.WrapError(err, oauth.AccessDenied)
	}

	return &oauth.CodeComponents{
		Key:                 s.CodeExchangeKey,
		CodeChallenge:       r.CodeChallenge,
		CodeChallengeMethod: r.CodeChallengeMethod,
		UserName:            userDetails.Name,
		UserEmail:           userDetails.Email,
	}, nil
}

func (s *DefaultLoginService) Redirect(cc *oauth.CodeComponents, r *oauth.Request) (string, oauth.Error) {
	payload, err := cc.ToPayload(s.EncryptSalt)
	if err != nil {
		return "", oauth.WrapError(err, oauth.InvalidRequest)
	}

	return fmt.Sprintf("%s?state=%s&code=%s", r.RedirectURI, r.State, payload), nil
}

func (s *DefaultLoginService) ValidateToken(components *oauth.CodeComponents, verifier string) (*oauth.TokenResponse, oauth.Error) {
	if components.CodeChallengeMethod != "S256" {
		return nil, oauth.WrapError(fmt.Errorf("code challenge method unsupported"), oauth.UnsupportedResponseType)
	}

	codeVerifierHash := sha256.Sum256([]byte(verifier))
	codeVerifierDecoded := base64.RawURLEncoding.EncodeToString(codeVerifierHash[:])

	if codeVerifierDecoded != components.CodeChallenge {
		return nil, oauth.WrapError(fmt.Errorf("code verification failed"), oauth.InvalidRequest)
	}

	t, err := s.JWT.Build(auth.User{
		Name:  components.UserName,
		Email: components.UserEmail,
	}, tokenExpirationInSeconds)
	if err != nil {
		return nil, oauth.WrapError(err, oauth.InvalidRequest)
	}

	return &oauth.TokenResponse{
		AccessToken:  t,
		TokenType:    "bearer",
		RefreshToken: "",
		ExpiresIn:    tokenExpirationInSeconds,
	}, nil
}
