package highwater

import (
	"testing"
)

const (
	EVENT_NAME = "testing"
	USERID     = "123-456-cc-2"
	TOKEN      = "a.fake.token.for.this"
)

func TestMock(t *testing.T) {

	p := make(map[string]string)

	p["one"] = "two"
	p["buckle"] = "my"
	p["shoe"] = "three ..."

	client := NewMock()

	client.PostServer(EVENT_NAME, TOKEN, p)

	client.PostThisUser(EVENT_NAME, TOKEN, p)

	client.PostWithUser(USERID, EVENT_NAME, TOKEN, p)
}

//log.Panic is called when not all required args are passed.
//This test fails if the panic and subseqent recover are not called
func TestMock_Fails(t *testing.T) {

	defer func() {
		if r := recover(); r == nil {
			t.Error("should have paniced")
		}
	}()

	p := make(map[string]string)

	p["one"] = "two"
	p["buckle"] = "my"
	p["shoe"] = "three ..."

	client := NewMock()

	client.PostServer("", TOKEN, p)

	client.PostThisUser(EVENT_NAME, "", p)

	client.PostWithUser("", EVENT_NAME, TOKEN, p)

	client.PostWithUser("", EVENT_NAME, TOKEN, nil)
}
