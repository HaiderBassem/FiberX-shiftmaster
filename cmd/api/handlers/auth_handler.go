package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"shiftmaster-backend/internal/config"
	"shiftmaster-backend/internal/middleware"
	"shiftmaster-backend/internal/models"
	"shiftmaster-backend/internal/service"
)

// AuthHandler handles authentication endpoints.
type AuthHandler struct {
	authService     *service.AuthService
	employeeService *service.EmployeeService
	jwtCfg          config.JWTConfig
}

func NewAuthHandler(authSvc *service.AuthService, empSvc *service.EmployeeService, jwtCfg config.JWTConfig) *AuthHandler {
	return &AuthHandler{
		authService:     authSvc,
		employeeService: empSvc,
		jwtCfg:          jwtCfg,
	}
}

type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type loginResponse struct {
	AccessToken  string      `json:"access_token"`
	RefreshToken string      `json:"refresh_token"`
	ExpiresIn    int         `json:"expires_in"`
	Employee     interface{} `json:"employee"`
}

// Login authenticates a user and returns JWT tokens.
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid request: " + err.Error()})
		return
	}

	emp, err := h.authService.Authenticate(c.Request.Context(), req.Email, req.Password, c.ClientIP())
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": err.Error()})
		return
	}

	accessToken, refreshToken, err := h.issueTokens(emp)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": loginResponse{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
			ExpiresIn:    h.jwtCfg.AccessExpireMin * 60,
			Employee:     emp,
		},
	})
}

// issueTokens mints a matched access/refresh pair from the employee's *current*
// database state. Callers must never build tokens from claims carried by an
// incoming token, or a role or department change would not take effect until the
// refresh token finally expired.
func (h *AuthHandler) issueTokens(emp *models.Employee) (accessToken string, refreshToken string, err error) {
	var deptID *string
	if emp.DepartmentID != nil {
		idStr := emp.DepartmentID.String()
		deptID = &idStr
	}

	accessToken, err = h.generateToken(emp, deptID, middleware.TokenTypeAccess, h.jwtCfg.AccessTokenDuration())
	if err != nil {
		return "", "", err
	}

	refreshToken, err = h.generateToken(emp, deptID, middleware.TokenTypeRefresh, h.jwtCfg.RefreshTokenDuration())
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

func (h *AuthHandler) generateToken(emp *models.Employee, departmentID *string, tokenType string, expiry time.Duration) (string, error) {
	now := time.Now()
	claims := middleware.Claims{
		EmployeeID:   emp.ID.String(),
		Email:        emp.Email,
		Role:         emp.Role,
		DepartmentID: departmentID,
		TokenType:    tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   emp.ID.String(),
			ExpiresAt: jwt.NewNumericDate(now.Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    h.jwtCfg.Issuer,
			ID:        uuid.New().String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(h.jwtCfg.Secret))
}

type changePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

// ChangePassword changes the authenticated user's password.
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req changePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid request: " + err.Error()})
		return
	}

	empIDStr, _ := c.Get("employee_id")
	empID, err := uuid.Parse(empIDStr.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "invalid token"})
		return
	}

	if err := h.authService.ChangePassword(c.Request.Context(), empID, req.OldPassword, req.NewPassword); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"message": "password changed successfully"}})
}

type resetPasswordRequest struct {
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

// ResetPassword allows a manager to reset an employee's password.
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	empID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid employee ID"})
		return
	}

	var req resetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid request: " + err.Error()})
		return
	}

	if err := h.authService.ResetPassword(c.Request.Context(), empID, req.NewPassword); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"message": fmt.Sprintf("password reset for employee %s", empID)}})
}

// Me returns the current authenticated user's profile.
func (h *AuthHandler) Me(c *gin.Context) {
	empIDStr, _ := c.Get("employee_id")
	empID, err := uuid.Parse(empIDStr.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "invalid token"})
		return
	}

	emp, err := h.employeeService.GetByID(c.Request.Context(), empID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "employee not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": emp})
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// Refresh validates a refresh token and issues new access + refresh tokens.
//
// The employee is re-read from the database on every refresh and the new tokens are
// signed from that current state. Authorization carried by the incoming token is
// treated as untrusted input: without this, a deactivated, locked-out or demoted
// employee would keep their former privileges until the refresh token expired.
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "refresh_token is required"})
		return
	}

	// Only a refresh token is accepted here; an access token is rejected.
	claims, err := middleware.ParseToken(req.RefreshToken, h.jwtCfg.Secret, h.jwtCfg.Issuer, middleware.TokenTypeRefresh)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "invalid or expired refresh token"})
		return
	}

	empID, err := uuid.Parse(claims.EmployeeID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "invalid token claims"})
		return
	}

	// Re-validate against current database state: existence, employment status and
	// authentication lock are all re-checked here, exactly as they are at login.
	emp, err := h.authService.GetAuthorizedEmployee(c.Request.Context(), empID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": err.Error()})
		return
	}

	accessToken, refreshToken, err := h.issueTokens(emp)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": loginResponse{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
			ExpiresIn:    h.jwtCfg.AccessExpireMin * 60,
			Employee:     emp,
		},
	})
}
