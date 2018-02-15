package shoreline

import (
	"testing"
)

func TestMock(t *testing.T) {

	const mockToken = "this is a token"

	client := NewMock(mockToken)

	if err := client.Start(); err != nil {
		t.Errorf("Failed start with error[%v]", err)
	}

	if tok := client.TokenProvide(); tok != mockToken {
		t.Errorf("Unexpected token[%s]", tok)
	}

	if usr, token, err := client.Login("billy", "howdy"); err != nil {
		t.Errorf("Failed start with error[%v]", err)
	} else {
		if usr == nil {
			t.Error("Should give us a fake usr details")
		}
		if token == "" {
			t.Error("Should give us a fake token")
		}
	}

	if checkedTd := client.CheckToken(mockToken); checkedTd == nil {
		t.Error("Should give us token data")
	}

	if checkedTd := client.CheckTokenForScopes("read:profile, write:profile", mockToken); checkedTd == nil {
		t.Error("Should give us token data")
	}

	if usr, _ := client.GetUser("billy@howdy.org", mockToken); usr == nil {
		t.Error("Should give us a mock user")
	}

	username := "name"
	password := "myN3wPw"
	user := UserUpdate{Username: &username, Emails: &[]string{"an@email.org"}, Password: &password}

	if err := client.UpdateUser("123", user, mockToken); err != nil {
		t.Error("Should return no error on success")
	}

	if sd, se := client.Signup("username", "password", "email@place.org"); sd == nil || se != nil {
		t.Errorf("Signup not return err[%s]", se.Error())
	}

}
