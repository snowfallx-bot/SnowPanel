package dto

type LoginRequest struct {
	Username string `json:"username" binding:"required,min=3,max=64"`
	Password string `json:"password" binding:"required,min=6,max=128"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required,min=6,max=128"`
	NewPassword     string `json:"new_password" binding:"required,min=8,max=128"`
}

type UserProfile struct {
	ID                 int64    `json:"id"`
	Username           string   `json:"username"`
	Email              string   `json:"email"`
	Status             int16    `json:"status"`
	Roles              []string `json:"roles"`
	Permissions        []string `json:"permissions"`
	MustChangePassword bool     `json:"must_change_password"`
}

type LoginResponse struct {
	AccessToken string      `json:"access_token"`
	TokenType   string      `json:"token_type"`
	ExpiresIn   int64       `json:"expires_in"`
	User        UserProfile `json:"user"`
}
