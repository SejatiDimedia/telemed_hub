package mapper

import (
	"github.com/timurdianradhasejati/telemed_hub/internal/auth/dto"
	"github.com/timurdianradhasejati/telemed_hub/internal/auth/model"
)

// ToUserResponse converts a User model and their assigned roles to a UserResponse DTO.
func ToUserResponse(user *model.User, roles []string) dto.UserResponse {
	return dto.UserResponse{
		ID:       user.ID.String(),
		Email:    user.Email,
		Roles:    roles,
		FullName: user.FullName,
	}
}

// ToRegisterResponse converts a User model and registration role to a RegisterResponse DTO.
func ToRegisterResponse(user *model.User, role string) dto.RegisterResponse {
	return dto.RegisterResponse{
		ID:    user.ID.String(),
		Email: user.Email,
		Role:  role,
	}
}
