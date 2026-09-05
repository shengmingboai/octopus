package op

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/charmbracelet/log"
	"github.com/shengmingboai/octopus/internal/db"
	"github.com/shengmingboai/octopus/internal/model"
	"gorm.io/gorm"
)

var userCache model.User
var userMu sync.RWMutex

func UserInit() error {
	userMu.Lock()
	defer userMu.Unlock()
	var user model.User
	if err := db.GetDB().First(&user).Error; err == nil {
		userCache = user
		return nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	user = model.User{Username: "admin", Password: "admin"}
	if err := user.HashPassword(); err != nil {
		return err
	}
	if err := db.GetDB().Create(&user).Error; err != nil {
		return err
	}
	userCache = user
	log.Infof("initial user: admin,password: admin")
	return nil
}

func UserChangePassword(oldPassword, newPassword string) error {
	userMu.Lock()
	defer userMu.Unlock()
	user := userCache
	if newPassword == "" {
		return fmt.Errorf("new password is required")
	}
	if err := user.ComparePassword(oldPassword); err != nil {
		return fmt.Errorf("incorrect old password: %w", err)
	}

	user.Password = newPassword
	if err := user.HashPassword(); err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	if err := db.GetDB().Model(&user).Update("password", user.Password).Error; err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}
	userCache = user
	return nil
}

func UserChangeUsername(newUsername string) error {
	userMu.Lock()
	defer userMu.Unlock()
	newUsername = strings.TrimSpace(newUsername)
	if newUsername == "" {
		return fmt.Errorf("new username is required")
	}
	if userCache.Username == newUsername {
		return fmt.Errorf("new username is the same as the old username")
	}
	user := userCache
	user.Username = newUsername
	if err := db.GetDB().Model(&user).Update("username", user.Username).Error; err != nil {
		return fmt.Errorf("failed to update username: %w", err)
	}
	userCache = user
	return nil
}

func UserVerify(username, password string) error {
	user := UserGet()
	if username != user.Username {
		return fmt.Errorf("incorrect username")
	}
	if err := user.ComparePassword(password); err != nil {
		return fmt.Errorf("incorrect password")
	}
	return nil
}

func UserGet() model.User {
	userMu.RLock()
	defer userMu.RUnlock()
	return userCache
}
