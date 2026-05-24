package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"shiftmaster-backend/internal/middleware"
	"shiftmaster-backend/internal/service"
)

// AuthHandler handles authentication endpoints.
type AuthHandler struct {
	authService     *service.AuthService
	employeeService *service.EmployeeService
	jwtSecret       string
	accessExpMin    int
	refreshExpDays  int
}

func NewAuthHandler(authSvc *service.AuthService, empSvc *service.EmployeeService, jwtSecret string, accessExpMin, refreshExpDays int) *AuthHandler {
	return &AuthHandler{
		authService:     authSvc,
		employeeService: empSvc,
		jwtSecret:       jwtSecret,
		accessExpMin:    accessExpMin,
		refreshExpDays:  refreshExpDays,
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

	emp, err := h.authService.Authenticate(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": err.Error()})
		return
	}

	var deptID *string
	if emp.DepartmentID != nil {
		idStr := emp.DepartmentID.String()
		deptID = &idStr
	}

	accessToken, err := h.generateToken(emp.ID.String(), emp.Email, emp.Role, deptID, time.Duration(h.accessExpMin)*time.Minute)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "failed to generate token"})
		return
	}

	refreshToken, err := h.generateToken(emp.ID.String(), emp.Email, emp.Role, deptID, time.Duration(h.refreshExpDays)*24*time.Hour)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "failed to generate refresh token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": loginResponse{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
			ExpiresIn:    h.accessExpMin * 60,
			Employee:     emp,
		},
	})
}

func (h *AuthHandler) generateToken(employeeID, email, role string, departmentID *string, expiry time.Duration) (string, error) {
	claims := middleware.Claims{
		EmployeeID:   employeeID,
		Email:        email,
		Role:         role,
		DepartmentID: departmentID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "shiftmaster-api",
			ID:        uuid.New().String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(h.jwtSecret))
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
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "refresh_token is required"})
		return
	}

	// Parse and validate the refresh token
	claims := &middleware.Claims{}
	token, err := jwt.ParseWithClaims(req.RefreshToken, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(h.jwtSecret), nil
	})
	if err != nil || !token.Valid {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "invalid or expired refresh token"})
		return
	}

	// Generate new tokens
	accessToken, err := h.generateToken(claims.EmployeeID, claims.Email, claims.Role, claims.DepartmentID, time.Duration(h.accessExpMin)*time.Minute)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "failed to generate access token"})
		return
	}

	refreshToken, err := h.generateToken(claims.EmployeeID, claims.Email, claims.Role, claims.DepartmentID, time.Duration(h.refreshExpDays)*24*time.Hour)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "failed to generate refresh token"})
		return
	}

	// Fetch employee data
	empID, err := uuid.Parse(claims.EmployeeID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "invalid token claims"})
		return
	}

	emp, err := h.employeeService.GetByID(c.Request.Context(), empID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "employee not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": loginResponse{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
			ExpiresIn:    h.accessExpMin * 60,
			Employee:     emp,
		},
	})
}
