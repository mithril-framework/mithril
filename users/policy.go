package users

import (
	"net/http"
	"strings"

	"github.com/mithril-framework/mithril/database/models"

	"golang.org/x/crypto/bcrypt"
)

const (
	minPasswordLen = 8
	maxPasswordLen = 72 // bcrypt silently ignores bytes beyond 72
)

// actor describes the caller of a user-management endpoint. All flags come from the database.
type actor struct {
	isSelf      bool // caller is the target user
	canChange   bool // caller holds users.change
	isSuperuser bool // caller is a superuser (DB)
}

type policyError struct {
	status int
	msg    string
}

func deny(status int, msg string) *policyError { return &policyError{status: status, msg: msg} }

func validatePassword(p string) *policyError {
	if len(p) < minPasswordLen {
		return deny(http.StatusBadRequest, "password must be at least 8 characters")
	}
	if len(p) > maxPasswordLen {
		return deny(http.StatusBadRequest, "password must be at most 72 bytes")
	}
	return nil
}

func validateEmail(e string) (string, *policyError) {
	e = strings.TrimSpace(e)
	at := strings.LastIndex(e, "@")
	if at <= 0 || at == len(e)-1 || strings.ContainsAny(e, " \t\r\n") {
		return "", deny(http.StatusBadRequest, "invalid email")
	}
	return e, nil
}

func hashPassword(p string) (string, *policyError) {
	h, err := bcrypt.GenerateFromPassword([]byte(p), bcrypt.DefaultCost)
	if err != nil {
		return "", deny(http.StatusInternalServerError, "internal error")
	}
	return string(h), nil
}

// buildNewUser validates a create request and returns the user to insert (password hashed).
func buildNewUser(req *createUserRequest, a actor) (*models.User, *policyError) {
	email, perr := validateEmail(req.Email)
	if perr != nil {
		return nil, perr
	}
	if perr := validatePassword(req.Password); perr != nil {
		return nil, perr
	}
	u := &models.User{
		Email:     email,
		FirstName: strings.TrimSpace(req.FirstName),
		LastName:  strings.TrimSpace(req.LastName),
		IsActive:  true,
	}
	if req.IsActive != nil {
		u.IsActive = *req.IsActive
	}
	if req.IsSuperuser != nil && *req.IsSuperuser {
		if !a.isSuperuser {
			return nil, deny(http.StatusForbidden, "only a superuser can create a superuser")
		}
		u.IsSuperuser = true
	}
	if u.PasswordHash, perr = hashPassword(req.Password); perr != nil {
		return nil, perr
	}
	return u, nil
}

// applyUserUpdate enforces who may change which field and returns the updated user.
func applyUserUpdate(req *updateUserRequest, target models.User, a actor) (models.User, *policyError) {
	if !a.isSelf && !a.canChange && !a.isSuperuser {
		return target, deny(http.StatusForbidden, "forbidden")
	}
	if target.IsSuperuser && !a.isSelf && !a.isSuperuser {
		return target, deny(http.StatusForbidden, "only a superuser can modify a superuser account")
	}

	if req.IsSuperuser != nil && *req.IsSuperuser != target.IsSuperuser {
		if !a.isSuperuser {
			return target, deny(http.StatusForbidden, "only a superuser can change superuser status")
		}
		if a.isSelf {
			return target, deny(http.StatusBadRequest, "you cannot change your own superuser status")
		}
		target.IsSuperuser = *req.IsSuperuser
	}
	if req.IsActive != nil && *req.IsActive != target.IsActive {
		if a.isSelf {
			return target, deny(http.StatusForbidden, "you cannot change your own active status")
		}
		target.IsActive = *req.IsActive
	}

	changesCredentials := req.Password != nil || (req.Email != nil && strings.TrimSpace(*req.Email) != target.Email)
	if a.isSelf && changesCredentials {
		// A stolen token alone must not be enough to take over the account permanently.
		if req.CurrentPassword == "" ||
			bcrypt.CompareHashAndPassword([]byte(target.PasswordHash), []byte(req.CurrentPassword)) != nil {
			return target, deny(http.StatusForbidden, "current_password is missing or incorrect")
		}
	}
	if req.Email != nil {
		email, perr := validateEmail(*req.Email)
		if perr != nil {
			return target, perr
		}
		target.Email = email
	}
	if req.Password != nil {
		if perr := validatePassword(*req.Password); perr != nil {
			return target, perr
		}
		hash, perr := hashPassword(*req.Password)
		if perr != nil {
			return target, perr
		}
		target.PasswordHash = hash
	}
	if req.FirstName != nil {
		target.FirstName = strings.TrimSpace(*req.FirstName)
	}
	if req.LastName != nil {
		target.LastName = strings.TrimSpace(*req.LastName)
	}
	return target, nil
}

// checkUserDelete enforces delete rules on top of the users.delete permission.
func checkUserDelete(target *models.User, a actor) *policyError {
	if a.isSelf {
		return deny(http.StatusBadRequest, "you cannot delete your own account here")
	}
	if target.IsSuperuser && !a.isSuperuser {
		return deny(http.StatusForbidden, "only a superuser can delete a superuser account")
	}
	return nil
}
