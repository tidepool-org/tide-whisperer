package auth

import (
	"context"
	"time"
)

type ClientMock struct {
	RestrictedToken *RestrictedToken
}

func NewMock() *ClientMock {
	now := time.Now()
	token := &RestrictedToken{
		ExpirationTime: now.Add(time.Hour * 24),
		CreatedTime:    now,
		ModifiedTime:   &now,
	}
	return &ClientMock{
		RestrictedToken: token,
	}
}

func (c *ClientMock) GetRestrictedToken(ctx context.Context, id string) (*RestrictedToken, error) {
	c.RestrictedToken.ID = id
	c.RestrictedToken.UserID = id
	return c.RestrictedToken, nil
}
