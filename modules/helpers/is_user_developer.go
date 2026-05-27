package helpers

import (
	"fmt"
	"slices"
	"zaglyt-tg/configs"
)

func IsUserDeveloper(userID int64) bool {
	config, err := configs.LoadConfig()
	if err != nil {
		fmt.Println(err)
		return false
	}

	if slices.Contains(config.Developers, userID) {
		return true
	}

	return false
}
