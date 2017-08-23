package shoreline

import (
	"log"
)

type ShorelineMockClient struct {
	ServerToken string
}

func NewMock(token string) *ShorelineMockClient {
	return &ShorelineMockClient{ServerToken: token}
}

func (client *ShorelineMockClient) Start() error {
	log.Println("Started mock shoreline client")
	return nil
}

func (client *ShorelineMockClient) Close() {
	log.Println("Close mock shoreline client")
}

func (client *ShorelineMockClient) Login(username, password string) (*UserData, string, error) {
	return &UserData{UserID: "123.456.789", UserName: username, Emails: []string{username}}, client.ServerToken, nil
}

func (client *ShorelineMockClient) Signup(username, password, email string) (*UserData, error) {
	return &UserData{UserID: "123.xxx.456", UserName: username, Emails: []string{email}}, nil
}

func (client *ShorelineMockClient) CheckToken(token string) *TokenData {
	return &TokenData{UserID: "987.654.321", IsServer: true}
}

func (client *ShorelineMockClient) TokenProvide() string {
	return client.ServerToken
}

func (client *ShorelineMockClient) GetUser(userID, token string) (*UserData, error) {
	return &UserData{UserID: userID, UserName: "From Mock", Emails: []string{userID}}, nil
}

func (client *ShorelineMockClient) UpdateUser(user UserUpdate, token string) error {
	return nil
}
