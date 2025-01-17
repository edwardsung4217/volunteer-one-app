package controllers

import (
	"bytes"
	"fmt"
	"github.com/VolunteerOne/volunteer-one-app/backend/middleware"
	"github.com/google/uuid"
	"net/http"
	"os"
	"time"

	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"

	"github.com/VolunteerOne/volunteer-one-app/backend/mocks"
	"github.com/VolunteerOne/volunteer-one-app/backend/models"
)

func getClaims() (jwt.Claims, jwt.Claims) {
	fakeAccessExpire := jwt.NewNumericDate(time.Now().Add(time.Minute * 15))
	fakeRefreshExpire := jwt.NewNumericDate(time.Now().Add(time.Hour * 24 * 30))
	accessTokenClaims := jwt.MapClaims{
		"sub":  uint(0),
		"exp":  fakeAccessExpire,
		"type": "access",
	}
	refreshTokenClaims := jwt.MapClaims{
		"sub":  uint(0),
		"exp":  fakeRefreshExpire,
		"type": "refresh",
	}
	return accessTokenClaims, refreshTokenClaims
}

// Tests when a good email and password occurs
func TestLoginController_Login_EmailFound(t *testing.T) {
	email := "test@user.com"
	password := "password"

	w := httptest.NewRecorder()
	// start new gin context to pass in
	c, _ := gin.CreateTestContext(w)

	c.AddParam("email", email)
	c.AddParam("password", password)

	// example user model to pass in empty
	var emptyUser models.Users
	var delegations models.Delegations

	// expected user model
	var user models.Users
	user.Email = email
	user.Password = password

	// setup mock
	mockService := new(mocks.LoginService)
	// mock the function
	mockService.On("FindUserFromEmail", email, emptyUser).Return(user, nil)
	mockService.On("CompareHashedAndUserPass", []byte(password), password).Return(nil)
	accessTokenClaims, refreshTokenClaims := getClaims()
	mockService.On("GenerateJWT", jwt.SigningMethodHS256, accessTokenClaims, "").Return("", nil)
	mockService.On("GenerateJWT", jwt.SigningMethodHS256, refreshTokenClaims, "").Return("", nil)
	mockService.On("SaveRefreshToken", uint(0), "", delegations).Return(nil)

	// run actual handler
	res := NewLoginController(mockService)
	res.Login(c)

	// check that everything happened as expected
	mockService.AssertExpectations(t)

	// Verify response code
	assert.Equal(t, 200, c.Writer.Status())
}

// Tests that an 502 is returned when an error is returned from the database
func TestLoginController_Login_RetrievalError(t *testing.T) {
	email := "test@user.com"
	password := "password"

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.AddParam("email", email)
	c.AddParam("password", password)

	var emptyUser models.Users

	var user models.Users
	user.Email = email
	user.Password = password

	mockService := new(mocks.LoginService)
	mockService.On("FindUserFromEmail", email, emptyUser).Return(user, fmt.Errorf("Arrrrr"))

	res := NewLoginController(mockService)
	res.Login(c)

	mockService.AssertExpectations(t)

	assert.Equal(t, 502, c.Writer.Status())
}

// Tests that the passed param password and db passwords are different
func TestLoginController_Login_PasswordsDontMatch(t *testing.T) {
	email := "test@user.com"
	password := "password"

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.AddParam("email", email)
	c.AddParam("password", "not right password")

	var emptyUser models.Users

	var user models.Users
	user.Email = email
	user.Password = password

	mockService := new(mocks.LoginService)
	mockService.On("FindUserFromEmail", email, emptyUser).Return(user, nil)
	mockService.On("CompareHashedAndUserPass", []byte(password), "not right password").Return(fmt.Errorf("error"))

	res := NewLoginController(mockService)
	res.Login(c)

	mockService.AssertExpectations(t)

	assert.Equal(t, 400, c.Writer.Status())
}

func TestLoginController_Login_JWTErrorAccess(t *testing.T) {
	// Verifies that a error is caught if JWT token is not successfully generated
	email := "test@user.com"
	password := "password"

	w := httptest.NewRecorder()
	// start new gin context to pass in
	c, _ := gin.CreateTestContext(w)

	c.AddParam("email", email)
	c.AddParam("password", password)

	// example user model to pass in empty
	var emptyUser models.Users

	// expected user model
	var user models.Users
	user.Email = email
	user.Password = password

	accessTokenClaim, _ := getClaims()

	// setup mock
	mockService := new(mocks.LoginService)
	// mock the function
	mockService.On("FindUserFromEmail", email, emptyUser).Return(user, nil)
	mockService.On("CompareHashedAndUserPass", []byte(password), password).Return(nil)
	mockService.On("GenerateJWT", jwt.SigningMethodHS256, accessTokenClaim, "").Return("", fmt.Errorf("error"))

	// run actual handler
	res := NewLoginController(mockService)
	res.Login(c)

	// check that everything happened as expected
	mockService.AssertExpectations(t)

	// Verify response code
	assert.Equal(t, http.StatusBadRequest, c.Writer.Status())
}

func TestLoginController_Login_JWTErrorRefresh(t *testing.T) {
	// Verifies that a error is caught if JWT token is not successfully generated
	email := "test@user.com"
	password := "password"

	w := httptest.NewRecorder()
	// start new gin context to pass in
	c, _ := gin.CreateTestContext(w)

	c.AddParam("email", email)
	c.AddParam("password", password)

	// example user model to pass in empty
	var emptyUser models.Users

	// expected user model
	var user models.Users
	user.Email = email
	user.Password = password

	accessTokenClaim, refreshTokenClaim := getClaims()

	// setup mock
	mockService := new(mocks.LoginService)
	// mock the function
	mockService.On("FindUserFromEmail", email, emptyUser).Return(user, nil)
	mockService.On("CompareHashedAndUserPass", []byte(password), password).Return(nil)
	mockService.On("GenerateJWT", jwt.SigningMethodHS256, accessTokenClaim, "").Return("", nil)
	mockService.On("GenerateJWT", jwt.SigningMethodHS256, refreshTokenClaim, "").Return("", fmt.Errorf("error"))

	// run actual handler
	res := NewLoginController(mockService)
	res.Login(c)

	// check that everything happened as expected
	mockService.AssertExpectations(t)

	// Verify response code
	assert.Equal(t, http.StatusBadRequest, c.Writer.Status())
}

func TestLoginController_Login_FailedSaveRefreshToken(t *testing.T) {
	// Verifies that a error is caught if JWT token is not successfully generated
	email := "test@user.com"
	password := "password"

	w := httptest.NewRecorder()
	// start new gin context to pass in
	c, _ := gin.CreateTestContext(w)

	c.AddParam("email", email)
	c.AddParam("password", password)

	// example user model to pass in empty
	var emptyUser models.Users
	var delegations models.Delegations

	// expected user model
	var user models.Users
	user.Email = email
	user.Password = password

	// setup mock
	mockService := new(mocks.LoginService)
	// mock the function
	mockService.On("FindUserFromEmail", email, emptyUser).Return(user, nil)
	mockService.On("CompareHashedAndUserPass", []byte(password), password).Return(nil)
	accessTokenClaims, refreshTokenClaims := getClaims()
	mockService.On("GenerateJWT", jwt.SigningMethodHS256, accessTokenClaims, "").Return("", nil)
	mockService.On("GenerateJWT", jwt.SigningMethodHS256, refreshTokenClaims, "").Return("", nil)
	mockService.On("SaveRefreshToken", uint(0), "", delegations).Return(fmt.Errorf("error"))

	// run actual handler
	res := NewLoginController(mockService)
	res.Login(c)

	// check that everything happened as expected
	mockService.AssertExpectations(t)

	// Verify response code
	assert.Equal(t, 400, c.Writer.Status())
}

func TestLoginController_SendEmailForPassReset_FailFindUser(t *testing.T) {
	w := httptest.NewRecorder()
	// start new gin context to pass in
	c, _ := gin.CreateTestContext(w)

	email := "test@email.com"
	c.AddParam("email", email)

	// setup mock
	mockService := new(mocks.LoginService)
	// mock the function
	var user models.Users
	mockService.On("FindUserFromEmail", email, user).Return(user, fmt.Errorf("error"))

	// run actual handler
	res := NewLoginController(mockService)
	res.SendEmailForPassReset(c)

	// check that everything happened as expected
	mockService.AssertExpectations(t)

	// Verify response code
	assert.Equal(t, 502, c.Writer.Status())
}

func TestLoginController_SendEmailForPassReset_FailSaveResetCode(t *testing.T) {
	w := httptest.NewRecorder()
	// start new gin context to pass in
	c, _ := gin.CreateTestContext(w)

	email := "test@email.com"
	c.AddParam("email", email)

	// setup mock
	mockService := new(mocks.LoginService)
	// mock the function
	var user models.Users
	fakeUUID := uuid.New()
	mockService.On("GenerateUUID").Return(fakeUUID)
	mockService.On("FindUserFromEmail", email, user).Return(user, nil)
	mockService.On("SaveResetCodeToUser", fakeUUID, user).Return(fmt.Errorf("error"))

	// run actual handler
	res := NewLoginController(mockService)
	res.SendEmailForPassReset(c)

	// check that everything happened as expected
	mockService.AssertExpectations(t)

	// Verify response code
	assert.Equal(t, http.StatusInternalServerError, c.Writer.Status())
}

func TestLoginController_SendEmailForPassReset_FailSendResetCodeEmail(t *testing.T) {
	w := httptest.NewRecorder()
	// start new gin context to pass in
	c, _ := gin.CreateTestContext(w)

	email := "test@email.com"
	c.AddParam("email", email)

	// setup mock
	mockService := new(mocks.LoginService)
	// mock the function
	var user models.Users
	fakeUUID := uuid.New()
	mockService.On("GenerateUUID").Return(fakeUUID)
	mockService.On("FindUserFromEmail", email, user).Return(user, nil)
	mockService.On("SaveResetCodeToUser", fakeUUID, user).Return(nil)
	mockService.On("SendResetCodeToEmail", "", fakeUUID.String()).Return(fmt.Errorf("error"))

	// run actual handler
	res := NewLoginController(mockService)
	res.SendEmailForPassReset(c)

	// check that everything happened as expected
	mockService.AssertExpectations(t)

	// Verify response code
	assert.Equal(t, http.StatusInternalServerError, c.Writer.Status())
}

func TestLoginController_SendEmailForPassReset_AllGood(t *testing.T) {
	w := httptest.NewRecorder()
	// start new gin context to pass in
	c, _ := gin.CreateTestContext(w)

	email := "test@email.com"
	c.AddParam("email", email)

	// setup mock
	mockService := new(mocks.LoginService)
	// mock the function
	var user models.Users
	fakeUUID := uuid.New()
	mockService.On("GenerateUUID").Return(fakeUUID)
	mockService.On("FindUserFromEmail", email, user).Return(user, nil)
	mockService.On("SaveResetCodeToUser", fakeUUID, user).Return(nil)
	mockService.On("SendResetCodeToEmail", "", fakeUUID.String()).Return(nil)

	// run actual handler
	res := NewLoginController(mockService)
	res.SendEmailForPassReset(c)

	// check that everything happened as expected
	mockService.AssertExpectations(t)

	// Verify response code
	assert.Equal(t, http.StatusOK, c.Writer.Status())
}
func TestLoginController_PasswordReset_CantParseUUID(t *testing.T) {
	// Checks that the FindUserEmail fails
	w := httptest.NewRecorder()
	// start new gin context to pass in
	c, _ := gin.CreateTestContext(w)

	email := "test@email.com"
	c.AddParam("email", email)
	c.AddParam("resetcode", "fake code")
	c.AddParam("newpassword", "pass")

	// setup mock
	mockService := new(mocks.LoginService)
	// mock the function
	u, _ := uuid.NewUUID()
	mockService.On("ParseUUID", "fake code").Return(u, fmt.Errorf("error"))

	// run actual handler
	res := NewLoginController(mockService)
	res.PasswordReset(c)

	// check that everything happened as expected
	mockService.AssertExpectations(t)

	// Verify response code
	assert.Equal(t, 500, c.Writer.Status())
}

func TestLoginController_PasswordReset_CantFindEmail(t *testing.T) {
	// Checks that the FindUserEmail fails
	w := httptest.NewRecorder()
	// start new gin context to pass in
	c, _ := gin.CreateTestContext(w)

	email := "test@email.com"
	c.AddParam("email", email)
	c.AddParam("resetcode", "fake code")
	c.AddParam("newpassword", "pass")

	// setup mock
	mockService := new(mocks.LoginService)
	// mock the function
	var user models.Users
	u, _ := uuid.NewUUID()
	mockService.On("ParseUUID", "fake code").Return(u, nil)
	mockService.On("FindUserFromEmail", email, user).Return(user, fmt.Errorf("error"))

	// run actual handler
	res := NewLoginController(mockService)
	res.PasswordReset(c)

	// check that everything happened as expected
	mockService.AssertExpectations(t)

	// Verify response code
	assert.Equal(t, 502, c.Writer.Status())
}

func TestLoginController_PasswordReset_CheckResetCodeFail(t *testing.T) {
	// Checks that if the reset codes aren't matching it fails
	w := httptest.NewRecorder()
	// start new gin context to pass in
	c, _ := gin.CreateTestContext(w)

	email := "test@email.com"
	resetCode := "2322db5b-b7f1-4ed6-9618-8662518a3c6e"
	c.AddParam("email", email)
	c.AddParam("resetcode", resetCode)
	c.AddParam("newpassword", "pass")

	// setup mock
	mockService := new(mocks.LoginService)
	// mock the function
	var user models.Users
	user.ResetCode, _ = uuid.Parse("00000000-0000-0000-0000-000000000000")

	u, _ := uuid.NewUUID()
	mockService.On("ParseUUID", resetCode).Return(u, nil)
	mockService.On("FindUserFromEmail", email, user).Return(user, nil)

	// run actual handler
	res := NewLoginController(mockService)
	res.PasswordReset(c)

	// check that everything happened as expected
	mockService.AssertExpectations(t)
	// Verify response code
	assert.Equal(t, 500, c.Writer.Status())
}

func TestLoginController_PasswordReset_ChangePasswordFail(t *testing.T) {
	// Checks that if the reset codes aren't matching it fails
	w := httptest.NewRecorder()
	// start new gin context to pass in
	c, _ := gin.CreateTestContext(w)

	email := "test@email.com"
	resetCode := "00000000-0000-0000-0000-000000000000"
	c.AddParam("email", email)
	c.AddParam("resetcode", resetCode)
	c.AddParam("newpassword", "pass")

	// setup mock
	mockService := new(mocks.LoginService)
	// mock the function
	var user models.Users
	user.ResetCode, _ = uuid.Parse(resetCode)

	mockService.On("ParseUUID", resetCode).Return(user.ResetCode, nil)
	mockService.On("FindUserFromEmail", email, user).Return(user, nil)
	mockService.On("HashPassword", []byte("pass")).Return([]byte("hashed pass"), nil)
	mockService.On("ChangePassword", []byte("hashed pass"), user).Return(fmt.Errorf("error"))

	// run actual handler
	res := NewLoginController(mockService)
	res.PasswordReset(c)

	// check that everything happened as expected
	mockService.AssertExpectations(t)
	// Verify response code
	assert.Equal(t, 500, c.Writer.Status())
}

func TestLoginController_PasswordReset_Success(t *testing.T) {
	// Checks that if the reset codes aren't matching it fails
	w := httptest.NewRecorder()
	// start new gin context to pass in
	c, _ := gin.CreateTestContext(w)

	email := "test@email.com"
	resetCode := "00000000-0000-0000-0000-000000000000"
	c.AddParam("email", email)
	c.AddParam("resetcode", resetCode)
	c.AddParam("newpassword", "pass")

	// setup mock
	mockService := new(mocks.LoginService)
	// mock the function
	var user models.Users
	user.ResetCode, _ = uuid.Parse(resetCode)

	mockService.On("ParseUUID", resetCode).Return(user.ResetCode, nil)
	mockService.On("FindUserFromEmail", email, user).Return(user, nil)
	mockService.On("HashPassword", []byte("pass")).Return([]byte("hashed pass"), nil)
	mockService.On("ChangePassword", []byte("hashed pass"), user).Return(nil)

	// run actual handler
	res := NewLoginController(mockService)
	res.PasswordReset(c)

	// check that everything happened as expected
	mockService.AssertExpectations(t)
	// Verify response code
	assert.Equal(t, 200, c.Writer.Status())
}

func TestLoginController_VerifyAcessToken(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	mockService := new(mocks.LoginService)
	res := NewLoginController(mockService)
	res.VerifyAccessToken(c)

	mockService.AssertExpectations(t)
	assert.Equal(t, 200, c.Writer.Status())
}

func TestLoginController_RefreshToken_NoTokenInHeader(t *testing.T) {
	// Tests that if there is no token field in the header an error isn't caught
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	mockService := new(mocks.LoginService)
	res := NewLoginController(mockService)
	req := httptest.NewRequest("POST", "/login/refresh", bytes.NewBuffer([]byte("")))
	c.Request = req
	res.RefreshToken(c)

	mockService.AssertExpectations(t)
	assert.Equal(t, 401, c.Writer.Status())
}

func TestLoginController_RefreshToken_FailedValidationAlg(t *testing.T) {
	// We sign with HMAC -> test if we sign with something like RSA that it breaks
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	mockService := new(mocks.LoginService)
	res := NewLoginController(mockService)
	req := httptest.NewRequest("POST", "/login/refresh", bytes.NewBuffer([]byte("")))

	// generate a bad signing jwt
	refreshExpire := jwt.NewNumericDate(time.Now().Add(time.Hour * 24))
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"sub":  0,
		"exp":  refreshExpire,
		"type": "refresh",
	})
	pem := []byte(`-----BEGIN RSA PRIVATE KEY-----
MIICWgIBAAKBgHBsap0ksx4hmQego6yldvuve7ZI0+rQ6uX5GPVTQXfIPwrlbtR8
+Chir3ALyQXa6GD8+8QCA5kJDi+2I2Un7JBncBPboXPAK92XG3ZZgYR11BApnDNS
Clsll0WBMKJJadWIhsJYI3kFZuwyIAZakx/h1/aqoBHkoP3A3cRqGhHbAgMBAAEC
gYBKD8I0f8bYJL4RfkwVInQ93h8buOKSoMr+cZl1lEFezbZqUTcwGJvKzyhQIhNu
HgZUTpT1TXZdTM/hspWiwChhHaxy+oJvf/T7XGWIxOgRhucymFKXsXIBjF0ypyiw
H7jcYrAOPIv8l9oWJZheAbWCBAZgF7m2h8KXFqNSDDuUcQJBAMylEMcBGB10QRYH
LtZ1wDuQ55+xDSlSCxNfB6hclk1dBdBt3YQ9zOX4O0TazYsz6Ij0uC9pOdCvtdDz
GS2WpZcCQQCMos5iqXo6vyJND3UqmL5dDWSGPDeOVF8F7epPZfzfsHoHXDAqEDEi
WH24u2YrlBXQEXU1vtRegVO9z12NAKZdAkAP5z//hxk9qLwqHxLHvczblC473cF9
FZAgyEDLF67igjkicndFgJv8vyaz+iEBEV6fzgzGOnIwwobpnwq03UEvAkApUcAf
frm5vuReDbeX706m2kN5qQGNoL5WaKNZ9pYIRrpjpTNFeIJnG+a8Otr23MhX3Hk6
dDnDFm47K0zKG7HFAkBAD9XTkjCdgQrRaURvRcX+M/L2cUej49KlaJr4Z6wKLPjL
48CNE12UkvPiYTC5TXLGoXczGlJE9KLKlNkjcl5F
-----END RSA PRIVATE KEY-----`)
	key, _ := jwt.ParseRSAPrivateKeyFromPEM(pem)
	signedToken, _ := refreshToken.SignedString(key)
	c.Request = req
	c.Request.Header["Token"] = []string{signedToken}

	c.Header("Token", signedToken)

	mockService.On("ValidateJWT", signedToken, "").Return(refreshToken, fmt.Errorf("error"))
	res.RefreshToken(c)

	mockService.AssertExpectations(t)
	assert.Equal(t, 401, c.Writer.Status())
}

func TestLoginController_RefreshToken_NotRefreshToken(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	mockService := new(mocks.LoginService)
	res := NewLoginController(mockService)
	req := httptest.NewRequest("POST", "/login/refresh", bytes.NewBuffer([]byte("")))

	// generate a bad signing jwt
	refreshExpire := jwt.NewNumericDate(time.Now().Add(time.Hour * 24))
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  0,
		"exp":  refreshExpire,
		"type": "not refresh",
	})
	os.Setenv("JWT_SECRET", "IO89UEYdRV$9tUA#jtM5hS!ch#hHqKXK")
	secret := os.Getenv("JWT_SECRET")
	signedToken, _ := refreshToken.SignedString([]byte(secret))
	c.Request = req
	c.Request.Header["Token"] = []string{signedToken}

	c.Header("Token", signedToken)

	fakeToken, _ := middleware.Validate(signedToken, secret)
	claims := jwt.MapClaims{}
	mockService.On("ValidateJWT", signedToken, secret).Return(fakeToken, nil)
	mockService.On("MapJWTClaims", *fakeToken).Return(claims, true)
	res.RefreshToken(c)

	mockService.AssertExpectations(t)
	assert.Equal(t, 401, c.Writer.Status())
}

func TestLoginController_RefreshToken_NotValidToken(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	mockService := new(mocks.LoginService)
	res := NewLoginController(mockService)
	req := httptest.NewRequest("POST", "/login/refresh", bytes.NewBuffer([]byte("")))

	// generate a bad signing jwt
	refreshExpire := jwt.NewNumericDate(time.Now().Add(time.Hour * 24))
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  0,
		"exp":  refreshExpire,
		"type": "not refresh",
	})
	os.Setenv("JWT_SECRET", "IO89UEYdRV$9tUA#jtM5hS!ch#hHqKXK")
	secret := os.Getenv("JWT_SECRET")
	signedToken, _ := refreshToken.SignedString([]byte(secret))
	c.Request = req
	c.Request.Header["Token"] = []string{signedToken}

	c.Header("Token", signedToken)

	fakeToken, _ := middleware.Validate(signedToken, secret)
	fakeToken.Valid = false
	mockService.On("ValidateJWT", signedToken, secret).Return(fakeToken, nil)
	res.RefreshToken(c)

	mockService.AssertExpectations(t)
	assert.Equal(t, 401, c.Writer.Status())
}

func TestLoginController_RefreshToken_CantFindRefreshToken(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	mockService := new(mocks.LoginService)
	res := NewLoginController(mockService)
	req := httptest.NewRequest("POST", "/login/refresh", bytes.NewBuffer([]byte("")))

	// generate a bad signing jwt
	refreshExpire := jwt.NewNumericDate(time.Now().Add(time.Hour * 24))
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  0,
		"exp":  refreshExpire,
		"type": "refresh",
	})
	os.Setenv("JWT_SECRET", "IO89UEYdRV$9tUA#jtM5hS!ch#hHqKXK")
	secret := os.Getenv("JWT_SECRET")
	signedToken, _ := refreshToken.SignedString([]byte(secret))
	c.Request = req
	c.Request.Header["Token"] = []string{signedToken}

	c.Header("Token", signedToken)

	var delegations models.Delegations

	fakeToken, _ := middleware.Validate(signedToken, secret)
	claims := jwt.MapClaims{}
	claims["type"] = "refresh"
	claims["sub"] = float64(0)
	claims["exp"] = fmt.Sprint(refreshExpire.Unix())
	mockService.On("ValidateJWT", signedToken, secret).Return(fakeToken, nil)
	mockService.On("MapJWTClaims", *fakeToken).Return(claims, true)
	mockService.On("FindRefreshToken", float64(0), delegations).Return(delegations, fmt.Errorf("error"))
	res.RefreshToken(c)

	mockService.AssertExpectations(t)
	assert.Equal(t, 401, c.Writer.Status())
}

func TestLoginController_RefreshToken_InvalidDelegationsToken(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	mockService := new(mocks.LoginService)
	res := NewLoginController(mockService)
	req := httptest.NewRequest("POST", "/login/refresh", bytes.NewBuffer([]byte("")))

	// generate a bad signing jwt
	refreshExpire := jwt.NewNumericDate(time.Now().Add(time.Hour * 24))
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  0,
		"exp":  refreshExpire,
		"type": "refresh",
	})
	os.Setenv("JWT_SECRET", "IO89UEYdRV$9tUA#jtM5hS!ch#hHqKXK")
	secret := os.Getenv("JWT_SECRET")
	signedToken, _ := refreshToken.SignedString([]byte(secret))
	c.Request = req
	c.Request.Header["Token"] = []string{signedToken}

	c.Header("Token", signedToken)

	var delegations models.Delegations
	var fakeDelegations models.Delegations
	fakeToken, _ := middleware.Validate(signedToken, secret)
	claims := jwt.MapClaims{}
	claims["type"] = "refresh"
	claims["sub"] = float64(0)
	claims["exp"] = fmt.Sprint(refreshExpire.Unix())

	mockService.On("ValidateJWT", signedToken, secret).Return(fakeToken, nil)
	mockService.On("MapJWTClaims", *fakeToken).Return(claims, true)
	mockService.On("FindRefreshToken", float64(0), delegations).Return(fakeDelegations, nil)
	mockService.On("DeleteRefreshToken", delegations).Return(fmt.Errorf("error"))

	res.RefreshToken(c)

	mockService.AssertExpectations(t)
	assert.Equal(t, 401, c.Writer.Status())
}

func TestLoginController_RefreshToken_InvalidDelegationsTokenNoError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	mockService := new(mocks.LoginService)
	res := NewLoginController(mockService)
	req := httptest.NewRequest("POST", "/login/refresh", bytes.NewBuffer([]byte("")))

	// generate a bad signing jwt
	refreshExpire := jwt.NewNumericDate(time.Now().Add(time.Hour * 24))
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  0,
		"exp":  refreshExpire,
		"type": "refresh",
	})
	os.Setenv("JWT_SECRET", "IO89UEYdRV$9tUA#jtM5hS!ch#hHqKXK")
	secret := os.Getenv("JWT_SECRET")
	signedToken, _ := refreshToken.SignedString([]byte(secret))
	c.Request = req
	c.Request.Header["Token"] = []string{signedToken}

	c.Header("Token", signedToken)

	var delegations models.Delegations
	var fakeDelegations models.Delegations
	fakeToken, _ := middleware.Validate(signedToken, secret)
	claims := jwt.MapClaims{}
	claims["type"] = "refresh"
	claims["sub"] = float64(0)
	claims["exp"] = fmt.Sprint(refreshExpire.Unix())

	mockService.On("ValidateJWT", signedToken, secret).Return(fakeToken, nil)
	mockService.On("MapJWTClaims", *fakeToken).Return(claims, true)
	mockService.On("FindRefreshToken", float64(0), delegations).Return(fakeDelegations, nil)
	mockService.On("DeleteRefreshToken", delegations).Return(nil)

	res.RefreshToken(c)

	mockService.AssertExpectations(t)
	assert.Equal(t, 401, c.Writer.Status())
}

func TestLoginController_RefreshToken_FaileGenerateJWTAccess(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	mockService := new(mocks.LoginService)
	res := NewLoginController(mockService)
	req := httptest.NewRequest("POST", "/login/refresh", bytes.NewBuffer([]byte("")))

	// generate a bad signing jwt
	accessExpire := jwt.NewNumericDate(time.Now().Add(time.Minute * 15))
	refreshExpire := jwt.NewNumericDate(time.Now().Add(time.Hour * 24))
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  0,
		"exp":  refreshExpire,
		"type": "refresh",
	})
	os.Setenv("JWT_SECRET", "IO89UEYdRV$9tUA#jtM5hS!ch#hHqKXK")
	secret := os.Getenv("JWT_SECRET")
	signedToken, _ := refreshToken.SignedString([]byte(secret))
	c.Request = req
	c.Request.Header["Token"] = []string{signedToken}

	c.Header("Token", signedToken)

	var delegations models.Delegations
	var fakeDelegations models.Delegations
	fakeDelegations.RefreshToken = signedToken

	fakeToken, _ := middleware.Validate(signedToken, secret)
	claims := jwt.MapClaims{}
	claims["type"] = "refresh"
	claims["sub"] = float64(0)
	claims["exp"] = fmt.Sprint(refreshExpire.Unix())

	mockService.On("ValidateJWT", signedToken, secret).Return(fakeToken, nil)
	mockService.On("MapJWTClaims", *fakeToken).Return(claims, true)
	mockService.On("FindRefreshToken", float64(0), delegations).Return(fakeDelegations, nil)
	accessTokenClaims, _ := getClaims()
	mockService.On("GenerateExpiresJWT").Return(accessExpire, refreshExpire)
	mockService.On("GenerateJWT", jwt.SigningMethodHS256, accessTokenClaims, secret).Return("", fmt.Errorf("error"))

	res.RefreshToken(c)

	mockService.AssertExpectations(t)
	assert.Equal(t, http.StatusBadRequest, c.Writer.Status())
}

func TestLoginController_RefreshToken_FaileGenerateJWTRefresh(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	mockService := new(mocks.LoginService)
	res := NewLoginController(mockService)
	req := httptest.NewRequest("POST", "/login/refresh", bytes.NewBuffer([]byte("")))

	// generate a bad signing jwt
	accessExpire := jwt.NewNumericDate(time.Now().Add(time.Minute * 15))
	refreshExpire := jwt.NewNumericDate(time.Now().Add(time.Hour * 24))
	claims := jwt.MapClaims{}
	claims["type"] = "refresh"
	claims["sub"] = float64(0)
	claims["exp"] = refreshExpire
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	os.Setenv("JWT_SECRET", "IO89UEYdRV$9tUA#jtM5hS!ch#hHqKXK")
	secret := os.Getenv("JWT_SECRET")
	signedToken, _ := refreshToken.SignedString([]byte(secret))
	c.Request = req
	c.Request.Header["Token"] = []string{signedToken}

	c.Header("Token", signedToken)

	var delegations models.Delegations
	var fakeDelegations models.Delegations
	fakeDelegations.RefreshToken = signedToken

	fakeToken, _ := middleware.Validate(signedToken, secret)
	mockService.On("ValidateJWT", signedToken, secret).Return(fakeToken, nil)
	mockService.On("MapJWTClaims", *fakeToken).Return(claims, true)
	mockService.On("FindRefreshToken", float64(0), delegations).Return(fakeDelegations, nil)

	mockService.On("GenerateExpiresJWT").Return(accessExpire, refreshExpire)

	fakeAccessClaims := jwt.MapClaims{
		"sub":  fakeDelegations.ID,
		"exp":  accessExpire,
		"type": "access",
	}

	mockService.On("GenerateJWT", jwt.SigningMethodHS256, fakeAccessClaims, secret).Return("", nil)

	fakeRefreshClaims := jwt.MapClaims{
		"sub":  fakeDelegations.ID,
		"exp":  refreshExpire,
		"type": "refresh",
	}

	mockService.On("GenerateJWT", jwt.SigningMethodHS256, fakeRefreshClaims, secret).Return("", fmt.Errorf("error"))

	res.RefreshToken(c)

	mockService.AssertExpectations(t)
	assert.Equal(t, http.StatusBadRequest, c.Writer.Status())
}

func TestLoginController_RefreshToken_FailedSaveRefreshToken(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	mockService := new(mocks.LoginService)
	res := NewLoginController(mockService)
	req := httptest.NewRequest("POST", "/login/refresh", bytes.NewBuffer([]byte("")))

	// generate a bad signing jwt
	accessExpire := jwt.NewNumericDate(time.Now().Add(time.Minute * 15))
	refreshExpire := jwt.NewNumericDate(time.Now().Add(time.Hour * 24))
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  0,
		"exp":  refreshExpire,
		"type": "refresh",
	})
	os.Setenv("JWT_SECRET", "IO89UEYdRV$9tUA#jtM5hS!ch#hHqKXK")
	secret := os.Getenv("JWT_SECRET")
	signedToken, _ := refreshToken.SignedString([]byte(secret))
	c.Request = req
	c.Request.Header["Token"] = []string{signedToken}

	c.Header("Token", signedToken)

	var delegations models.Delegations
	var fakeDelegations models.Delegations
	fakeDelegations.RefreshToken = signedToken

	fakeToken, _ := middleware.Validate(signedToken, secret)
	claims := jwt.MapClaims{}
	claims["type"] = "refresh"
	claims["sub"] = float64(0)
	claims["exp"] = fmt.Sprint(refreshExpire.Unix())

	mockService.On("ValidateJWT", signedToken, secret).Return(fakeToken, nil)
	mockService.On("MapJWTClaims", *fakeToken).Return(claims, true)
	mockService.On("FindRefreshToken", float64(0), delegations).Return(fakeDelegations, nil)
	w.Code = http.StatusOK
	mockService.On("GenerateExpiresJWT").Return(accessExpire, refreshExpire)

	fakeAccessClaims := jwt.MapClaims{
		"sub":  fakeDelegations.ID,
		"exp":  accessExpire,
		"type": "access",
	}

	mockService.On("GenerateJWT", jwt.SigningMethodHS256, fakeAccessClaims, secret).Return("", nil)

	fakeRefreshClaims := jwt.MapClaims{
		"sub":  fakeDelegations.ID,
		"exp":  refreshExpire,
		"type": "refresh",
	}

	mockService.On("GenerateJWT", jwt.SigningMethodHS256, fakeRefreshClaims, secret).Return("", nil)

	mockService.On("SaveRefreshToken", uint(0), "", fakeDelegations).Return(fmt.Errorf("error"))

	res.RefreshToken(c)

	mockService.AssertExpectations(t)
	assert.Equal(t, http.StatusInternalServerError, c.Writer.Status())
}

func TestLoginController_RefreshToken_FailMapClaims(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	mockService := new(mocks.LoginService)
	res := NewLoginController(mockService)
	req := httptest.NewRequest("POST", "/login/refresh", bytes.NewBuffer([]byte("")))

	// generate a bad signing jwt
	refreshExpire := jwt.NewNumericDate(time.Now().Add(time.Hour * 24))
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  0,
		"exp":  refreshExpire,
		"type": "refresh",
	})
	os.Setenv("JWT_SECRET", "IO89UEYdRV$9tUA#jtM5hS!ch#hHqKXK")
	secret := os.Getenv("JWT_SECRET")
	signedToken, _ := refreshToken.SignedString([]byte(secret))
	c.Request = req
	c.Request.Header["Token"] = []string{signedToken}

	c.Header("Token", signedToken)

	var fakeDelegations models.Delegations
	fakeDelegations.RefreshToken = signedToken

	fakeToken, _ := middleware.Validate(signedToken, secret)
	claims := jwt.MapClaims{}
	claims["type"] = "refresh"
	claims["sub"] = float64(0)
	claims["exp"] = fmt.Sprint(refreshExpire.Unix())

	mockService.On("ValidateJWT", signedToken, secret).Return(fakeToken, nil)
	mockService.On("MapJWTClaims", *fakeToken).Return(claims, false)
	res.RefreshToken(c)

	mockService.AssertExpectations(t)
	assert.Equal(t, http.StatusInternalServerError, c.Writer.Status())
}

func TestLoginController_RefreshToken_ValidToken(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	mockService := new(mocks.LoginService)
	res := NewLoginController(mockService)
	req := httptest.NewRequest("POST", "/login/refresh", bytes.NewBuffer([]byte("")))

	// generate a bad signing jwt
	accessExpire := jwt.NewNumericDate(time.Now().Add(time.Minute * 15))
	refreshExpire := jwt.NewNumericDate(time.Now().Add(time.Hour * 24))
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  0,
		"exp":  refreshExpire,
		"type": "refresh",
	})
	os.Setenv("JWT_SECRET", "IO89UEYdRV$9tUA#jtM5hS!ch#hHqKXK")
	secret := os.Getenv("JWT_SECRET")
	signedToken, _ := refreshToken.SignedString([]byte(secret))
	c.Request = req
	c.Request.Header["Token"] = []string{signedToken}

	c.Header("Token", signedToken)

	var delegations models.Delegations
	var fakeDelegations models.Delegations
	fakeDelegations.RefreshToken = signedToken

	fakeToken, _ := middleware.Validate(signedToken, secret)
	claims := jwt.MapClaims{}
	claims["type"] = "refresh"
	claims["sub"] = float64(0)
	claims["exp"] = fmt.Sprint(refreshExpire.Unix())

	mockService.On("ValidateJWT", signedToken, secret).Return(fakeToken, nil)
	mockService.On("MapJWTClaims", *fakeToken).Return(claims, true)
	mockService.On("FindRefreshToken", float64(0), delegations).Return(fakeDelegations, nil)
	w.Code = http.StatusOK

	mockService.On("GenerateExpiresJWT").Return(accessExpire, refreshExpire)

	fakeAccessClaims := jwt.MapClaims{
		"sub":  fakeDelegations.ID,
		"exp":  accessExpire,
		"type": "access",
	}

	mockService.On("GenerateJWT", jwt.SigningMethodHS256, fakeAccessClaims, secret).Return("", nil)

	fakeRefreshClaims := jwt.MapClaims{
		"sub":  fakeDelegations.ID,
		"exp":  refreshExpire,
		"type": "refresh",
	}

	mockService.On("GenerateJWT", jwt.SigningMethodHS256, fakeRefreshClaims, secret).Return("", nil)
	mockService.On("SaveRefreshToken", uint(0), "", fakeDelegations).Return(nil)

	res.RefreshToken(c)

	mockService.AssertExpectations(t)
	assert.Equal(t, http.StatusOK, c.Writer.Status())
}
