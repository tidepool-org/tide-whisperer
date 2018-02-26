package clients

import (
	"reflect"
	"testing"
)

//The purpose of this test is to ensure you canreply on the mocks

const USERID, GROUPID = "123user", "456group"

func makeExpectedPermissons() Permissions {
	return Permissions{USERID: Allowed}
}

func makeExpectedUsersPermissions() UsersPermissions {
	return UsersPermissions{GROUPID: Permissions{GROUPID: Allowed}}
}

func TestGatekeeperMock_UserInGroup(t *testing.T) {

	expected := makeExpectedPermissons()

	gkc := NewGatekeeperMock(nil, nil)

	if perms, err := gkc.UserInGroup(USERID, GROUPID); err != nil {
		t.Fatal("No error should be returned")
	} else if !reflect.DeepEqual(perms, expected) {
		t.Fatalf("Perms where [%v] but expected [%v]", perms, expected)
	}
}

func TestGatekeeperMock_UsersInGroup(t *testing.T) {

	expected := makeExpectedUsersPermissions()

	gkc := NewGatekeeperMock(nil, nil)

	if perms, err := gkc.UsersInGroup(GROUPID); err != nil {
		t.Fatal("No error should be returned")
	} else if !reflect.DeepEqual(perms, expected) {
		t.Fatalf("Perms were [%v] but expected [%v]", perms, expected)
	}
}

func TestGatekeeperMock_SetPermissions(t *testing.T) {

	gkc := NewGatekeeperMock(nil, nil)

	expected := makeExpectedPermissons()

	if perms, err := gkc.SetPermissions(USERID, GROUPID, expected); err != nil {
		t.Fatal("No error should be returned")
	} else if !reflect.DeepEqual(perms, expected) {
		t.Fatalf("Perms where [%v] but expected [%v]", perms, expected)

	}
}

func TestSeagullMock_GetCollection(t *testing.T) {

	sc := NewSeagullMock()
	var col struct{ Something string }

	sc.GetCollection("123.456", "stuff", &col)

	if col.Something != "anit no thing" {
		t.Error("Should have given mocked collection")
	}
}

func TestSeagullMock_GetPrivatePair(t *testing.T) {
	sc := NewSeagullMock()

	if pp := sc.GetPrivatePair("123.456", "Stuff"); pp == nil {
		t.Error("Should give us mocked private pair")
	}

}
