package repository

import (
	"context"
	"errors"

	"github.com/snowfallx-bot/SnowPanel/backend/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UserRepository interface {
	Count(ctx context.Context) (int64, error)
	Create(ctx context.Context, user *model.User) error
	GetByUsername(ctx context.Context, username string) (*model.User, error)
	GetByID(ctx context.Context, id int64) (*model.User, error)
	EnsureRBACDefaults(ctx context.Context) error
	EnsureUserRoleBySlug(ctx context.Context, userID int64, roleSlug string) error
	GetRolesAndPermissions(ctx context.Context, userID int64) ([]string, []string, error)
}

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.User{}).Count(&count).Error
	return count, err
}

func (r *userRepository) Create(ctx context.Context, user *model.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *userRepository) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) GetByID(ctx context.Context, id int64) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).First(&user, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) EnsureRBACDefaults(ctx context.Context) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		roles := []model.Role{
			{Name: "Super Admin", Slug: "super_admin", Description: "Full access to all resources"},
			{Name: "Operator", Slug: "operator", Description: "Limited operations access"},
		}
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "slug"}},
			DoNothing: true,
		}).Create(&roles).Error; err != nil {
			return err
		}

		type permissionSeed struct {
			Name        string
			Description string
		}
		permissionSeeds := []permissionSeed{
			{Name: "dashboard.read", Description: "Read dashboard overview"},
			{Name: "files.read", Description: "Read files"},
			{Name: "files.write", Description: "Write files"},
			{Name: "services.read", Description: "Read service states"},
			{Name: "services.manage", Description: "Manage services"},
			{Name: "docker.read", Description: "Read docker resources"},
			{Name: "docker.manage", Description: "Manage docker resources"},
			{Name: "cron.read", Description: "Read cron jobs"},
			{Name: "cron.manage", Description: "Manage cron jobs"},
			{Name: "audit.read", Description: "Read audit logs"},
			{Name: "tasks.read", Description: "Read tasks"},
			{Name: "tasks.manage", Description: "Manage tasks"},
		}

		permissions := make([]model.Permission, 0, len(permissionSeeds))
		for _, seed := range permissionSeeds {
			permissions = append(permissions, model.Permission{
				Name:        seed.Name,
				Description: seed.Description,
			})
		}
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "name"}},
			DoNothing: true,
		}).Create(&permissions).Error; err != nil {
			return err
		}

		var roleRecords []model.Role
		if err := tx.Where("slug IN ?", []string{"super_admin", "operator"}).Find(&roleRecords).Error; err != nil {
			return err
		}
		roleIDs := map[string]int64{}
		for _, role := range roleRecords {
			roleIDs[role.Slug] = role.ID
		}

		var permissionRecords []model.Permission
		permissionNames := make([]string, 0, len(permissionSeeds))
		for _, seed := range permissionSeeds {
			permissionNames = append(permissionNames, seed.Name)
		}
		if err := tx.Where("name IN ?", permissionNames).Find(&permissionRecords).Error; err != nil {
			return err
		}
		permissionIDs := map[string]int64{}
		for _, permission := range permissionRecords {
			permissionIDs[permission.Name] = permission.ID
		}

		superAdminPermissions := permissionNames
		operatorPermissions := []string{
			"dashboard.read",
			"files.read",
		}

		rolePermissions := make([]model.RolePermission, 0, len(superAdminPermissions)+len(operatorPermissions))
		for _, permissionName := range superAdminPermissions {
			roleID, roleOK := roleIDs["super_admin"]
			permissionID, permOK := permissionIDs[permissionName]
			if roleOK && permOK {
				rolePermissions = append(rolePermissions, model.RolePermission{
					RoleID:       roleID,
					PermissionID: permissionID,
				})
			}
		}
		for _, permissionName := range operatorPermissions {
			roleID, roleOK := roleIDs["operator"]
			permissionID, permOK := permissionIDs[permissionName]
			if roleOK && permOK {
				rolePermissions = append(rolePermissions, model.RolePermission{
					RoleID:       roleID,
					PermissionID: permissionID,
				})
			}
		}
		if len(rolePermissions) > 0 {
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "role_id"}, {Name: "permission_id"}},
				DoNothing: true,
			}).Create(&rolePermissions).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

func (r *userRepository) EnsureUserRoleBySlug(ctx context.Context, userID int64, roleSlug string) error {
	var role model.Role
	err := r.db.WithContext(ctx).
		Where("slug = ?", roleSlug).
		First(&role).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}

	userRole := model.UserRole{
		UserID: userID,
		RoleID: role.ID,
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "role_id"}},
		DoNothing: true,
	}).Create(&userRole).Error
}

func (r *userRepository) GetRolesAndPermissions(ctx context.Context, userID int64) ([]string, []string, error) {
	var roles []string
	if err := r.db.WithContext(ctx).
		Model(&model.Role{}).
		Distinct("roles.slug").
		Joins("JOIN user_roles ON user_roles.role_id = roles.id").
		Where("user_roles.user_id = ?", userID).
		Order("roles.slug ASC").
		Pluck("roles.slug", &roles).Error; err != nil {
		return nil, nil, err
	}

	var permissions []string
	if err := r.db.WithContext(ctx).
		Table("permissions").
		Distinct("permissions.name").
		Joins("JOIN role_permissions ON role_permissions.permission_id = permissions.id").
		Joins("JOIN user_roles ON user_roles.role_id = role_permissions.role_id").
		Where("user_roles.user_id = ?", userID).
		Order("permissions.name ASC").
		Pluck("permissions.name", &permissions).Error; err != nil {
		return nil, nil, err
	}

	return roles, permissions, nil
}
