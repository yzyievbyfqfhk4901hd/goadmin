package config

import (
	"encoding/json"
	"io/ioutil"
	"log"
)

type Config struct {
	BotToken        string  `json:"bot_token"`
	AuthorizedUsers []int64 `json:"authorized_users"`
}

func LoadConfig() (*Config, error) {
	data, err := ioutil.ReadFile("secrets.json")
	if err != nil {
		return nil, err
	}

	var config Config
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

func (c *Config) IsAuthorized(userID int64) bool {
	for _, authorizedID := range c.AuthorizedUsers {
		if userID == authorizedID {
			return true
		}
	}
	return false
}

func (c *Config) ValidateConfig() error {
	if c.BotToken == "" || c.BotToken == "YOUR_BOT_TOKEN_HERE" {
		log.Fatal("Please set a valid bot token in secrets.json")
	}

	if len(c.AuthorizedUsers) == 0 {
		log.Fatal("Please add at least one authorized user ID in secrets.json")
	}

	return nil
}
