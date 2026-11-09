package auth

import (
	"reflect"
	"time"

	jwtware "github.com/gofiber/contrib/v3/jwt"
	"github.com/gofiber/fiber/v3"
	"github.com/mithril-framework/mithril/acl"
	"github.com/mithril-framework/mithril/api"
	"github.com/mithril-framework/mithril/openapi"
)

// LoginRequest is the body of POST /auth/login.
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
	Remember bool   `json:"remember" doc:"Also set HTTP-only cookies"`
}

// RegisterRequest is the body of POST /auth/register.
type RegisterRequest struct {
	Email     string `json:"email" validate:"required,email"`
	Password  string `json:"password" validate:"required,min=8,max=72"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

// RefreshRequest is the body of POST /auth/refresh (the refresh_token cookie also works).
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// TokenPair is returned by login and refresh.
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	TokenType    string    `json:"token_type" example:"Bearer"`
}

// Mount registers the /auth routes; Public: register, login, refresh and the not-yet-implemented
// password/OTP endpoints. Protected (access token required): logout, me, 2FA.
// hs returns the current handlers (they are rebuilt when the database (re)connects).
func Mount(app fiber.Router, hs func() *Handlers, jwtCfg jwtware.Config, doc *openapi.Document) {
	g := app.Group("/auth")
	h := lazy{hs}
	g.Post("/register", h.Register)
	g.Post("/login", h.Login)
	g.Post("/forgot-password", h.ForgotPassword)
	g.Post("/reset-password", h.ResetPassword)
	g.Post("/send-otp", h.SendOTP)
	g.Post("/verify-otp", h.VerifyOTP)
	// Refresh is public: it is called exactly when the access token has expired, and it
	// authenticates with the refresh token itself (type=refresh, verified in the handler).
	g.Post("/refresh", h.Refresh)

	g.Use(jwtware.New(jwtCfg))
	g.Use(acl.JWTClaimsMiddleware()) // rejects refresh tokens used as access tokens

	g.Post("/logout", h.Logout)
	g.Get("/me", h.Me)
	g.Post("/enable-2fa", h.Enable2FA)
	g.Post("/verify-2fa", h.Verify2FA)

	if doc != nil {
		document(doc)
	}
}

func document(d *openapi.Document) {
	js := func(s *openapi.Schema) map[string]openapi.MediaType {
		return map[string]openapi.MediaType{"application/json": {Schema: s}}
	}
	body := func(v any) *openapi.RequestBody {
		return &openapi.RequestBody{Required: true, Content: js(d.SchemaForType(reflect.TypeOf(v)))}
	}
	errS := d.SchemaForType(reflect.TypeOf(api.Error{}))
	tokens := d.SchemaForType(reflect.TypeOf(TokenPair{}))
	tags := []string{"auth"}
	ok := func(desc string, s *openapi.Schema, errs ...string) map[string]openapi.Response {
		m := map[string]openapi.Response{"200": {Description: desc, Content: js(s)}}
		for _, e := range errs {
			m[e] = openapi.Response{Description: "Error", Content: js(errS)}
		}
		return m
	}
	d.AddOperation("POST", "/auth/login", openapi.Operation{OperationID: "login", Summary: "Log in with email and password", Tags: tags,
		RequestBody: body(LoginRequest{}), Responses: ok("Tokens and user", &openapi.Schema{Type: "object", Properties: map[string]*openapi.Schema{
			"success": {Type: "boolean"},
			"data":    {Type: "object", Properties: map[string]*openapi.Schema{"tokens": tokens, "user": {Type: "object"}}},
		}}, "401", "403")})
	d.AddOperation("POST", "/auth/register", openapi.Operation{OperationID: "register", Summary: "Register (only when ENABLE_REGISTER=true)", Tags: tags,
		RequestBody: body(RegisterRequest{}), Responses: map[string]openapi.Response{
			"201": {Description: "Created"}, "409": {Description: "Email exists", Content: js(errS)}, "503": {Description: "Registration disabled", Content: js(errS)}}})
	d.AddOperation("POST", "/auth/refresh", openapi.Operation{OperationID: "refresh", Summary: "Exchange a refresh token for a new token pair", Tags: tags,
		RequestBody: body(RefreshRequest{}), Responses: ok("New tokens", &openapi.Schema{Type: "object", Properties: map[string]*openapi.Schema{
			"success": {Type: "boolean"}, "data": tokens}}, "401")})
	d.AddOperation("GET", "/auth/me", openapi.Operation{OperationID: "me", Summary: "Current user from the access token", Tags: tags,
		Security: openapi.BearerAuth, Responses: ok("Current user", &openapi.Schema{Type: "object"}, "401")})
	d.AddOperation("POST", "/auth/logout", openapi.Operation{OperationID: "logout", Summary: "Clear auth cookies", Tags: tags,
		Security: openapi.BearerAuth, Responses: ok("Logged out", &openapi.Schema{Type: "object"}, "401")})
}

// lazy forwards to the current Handlers so routes can be registered before the DB is connected.
type lazy struct{ get func() *Handlers }

func (l lazy) Register(c fiber.Ctx) error       { return l.get().Register(c) }
func (l lazy) Login(c fiber.Ctx) error          { return l.get().Login(c) }
func (l lazy) ForgotPassword(c fiber.Ctx) error { return l.get().ForgotPassword(c) }
func (l lazy) ResetPassword(c fiber.Ctx) error  { return l.get().ResetPassword(c) }
func (l lazy) SendOTP(c fiber.Ctx) error        { return l.get().SendOTP(c) }
func (l lazy) VerifyOTP(c fiber.Ctx) error      { return l.get().VerifyOTP(c) }
func (l lazy) Refresh(c fiber.Ctx) error        { return l.get().Refresh(c) }
func (l lazy) Logout(c fiber.Ctx) error         { return l.get().Logout(c) }
func (l lazy) Me(c fiber.Ctx) error             { return l.get().Me(c) }
func (l lazy) Enable2FA(c fiber.Ctx) error      { return l.get().Enable2FA(c) }
func (l lazy) Verify2FA(c fiber.Ctx) error      { return l.get().Verify2FA(c) }
