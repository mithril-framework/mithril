package users

import (
	"testing"

	"github.com/mithril-framework/mithril/database/models"
	"golang.org/x/crypto/bcrypt"
)

func ptr[T any](v T) *T { return &v }

func mkUser(t *testing.T, super bool) models.User {
	t.Helper()
	h, err := bcrypt.GenerateFromPassword([]byte("old-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	return models.User{Email: "u@x.com", PasswordHash: string(h), FirstName: "F", LastName: "L", IsActive: true, IsSuperuser: super}
}

var (
	selfPlain  = actor{isSelf: true}
	changer    = actor{canChange: true}
	superActor = actor{isSuperuser: true, canChange: true}
)

func TestCreateCannotMassAssignSuperuser(t *testing.T) {
	req := &createUserRequest{Email: "n@x.com", Password: "longenough", IsSuperuser: ptr(true)}
	if _, perr := buildNewUser(req, changer); perr == nil || perr.status != 403 {
		t.Fatalf("users.add holder created a superuser: %+v", perr)
	}
	u, perr := buildNewUser(req, superActor)
	if perr != nil || !u.IsSuperuser {
		t.Fatalf("superuser should be able to create superuser: %+v", perr)
	}
	if u.PasswordHash == "longenough" || bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte("longenough")) != nil {
		t.Fatal("password must be stored as a bcrypt hash")
	}
}

func TestCreateValidates(t *testing.T) {
	if _, perr := buildNewUser(&createUserRequest{Email: "bad", Password: "longenough"}, superActor); perr == nil {
		t.Fatal("invalid email accepted")
	}
	if _, perr := buildNewUser(&createUserRequest{Email: "a@b.c", Password: "short"}, superActor); perr == nil {
		t.Fatal("short password accepted")
	}
}

func TestUsersChangeCannotEscalate(t *testing.T) {
	target := mkUser(t, false)
	if _, perr := applyUserUpdate(&updateUserRequest{IsSuperuser: ptr(true)}, target, changer); perr == nil || perr.status != 403 {
		t.Fatalf("users.change granted superuser: %+v", perr)
	}
	selfChanger := actor{isSelf: true, canChange: true}
	if _, perr := applyUserUpdate(&updateUserRequest{IsSuperuser: ptr(true)}, target, selfChanger); perr == nil {
		t.Fatal("users.change holder made themselves superuser")
	}
}

func TestUsersChangeCannotTouchSuperuser(t *testing.T) {
	target := mkUser(t, true)
	if _, perr := applyUserUpdate(&updateUserRequest{Password: ptr("new-password")}, target, changer); perr == nil || perr.status != 403 {
		t.Fatalf("users.change reset a superuser's password: %+v", perr)
	}
	if _, perr := applyUserUpdate(&updateUserRequest{Password: ptr("new-password")}, target, superActor); perr != nil {
		t.Fatalf("superuser should manage superusers: %+v", perr)
	}
}

func TestSelfServiceRestrictions(t *testing.T) {
	target := mkUser(t, false)
	if _, perr := applyUserUpdate(&updateUserRequest{IsActive: ptr(false)}, target, selfPlain); perr == nil {
		t.Fatal("self changed own is_active")
	}
	if _, perr := applyUserUpdate(&updateUserRequest{Password: ptr("new-password")}, target, selfPlain); perr == nil {
		t.Fatal("self changed password without current_password")
	}
	if _, perr := applyUserUpdate(&updateUserRequest{Email: ptr("evil@x.com"), CurrentPassword: "wrong"}, target, selfPlain); perr == nil {
		t.Fatal("self changed email with wrong current_password")
	}
	got, perr := applyUserUpdate(&updateUserRequest{Password: ptr("new-password"), CurrentPassword: "old-password"}, target, selfPlain)
	if perr != nil || bcrypt.CompareHashAndPassword([]byte(got.PasswordHash), []byte("new-password")) != nil {
		t.Fatalf("self password change with current_password failed: %+v", perr)
	}
	if _, perr := applyUserUpdate(&updateUserRequest{}, target, actor{}); perr == nil {
		t.Fatal("unprivileged non-self update allowed")
	}
}

func TestPartialUpdateKeepsOmittedFields(t *testing.T) {
	target := mkUser(t, false)
	got, perr := applyUserUpdate(&updateUserRequest{FirstName: ptr("New")}, target, selfPlain)
	if perr != nil {
		t.Fatal(perr.msg)
	}
	if got.FirstName != "New" || got.LastName != "L" || !got.IsActive || got.Email != target.Email || got.PasswordHash != target.PasswordHash {
		t.Fatalf("omitted fields changed: %+v", got)
	}
}

func TestSuperuserCannotDemoteSelf(t *testing.T) {
	target := mkUser(t, true)
	if _, perr := applyUserUpdate(&updateUserRequest{IsSuperuser: ptr(false)}, target, actor{isSelf: true, isSuperuser: true}); perr == nil {
		t.Fatal("superuser removed own superuser flag (lockout risk)")
	}
}

func TestDeleteRules(t *testing.T) {
	su := mkUser(t, true)
	if perr := checkUserDelete(&su, actor{canChange: true}); perr == nil {
		t.Fatal("non-superuser deleted a superuser")
	}
	if perr := checkUserDelete(&su, superActor); perr != nil {
		t.Fatalf("superuser delete denied: %+v", perr)
	}
	if perr := checkUserDelete(&su, actor{isSelf: true, isSuperuser: true}); perr == nil {
		t.Fatal("self delete allowed")
	}
}
