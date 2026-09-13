package config

import (
	"sync"
	"time"
)

type SupportConfig struct {
	BotEnabled              bool          `json:"bot_enabled"`
	BotName                 string        `json:"bot_name"`
	BotWelcomeMessage       string        `json:"bot_welcome_message"`
	AgentAcceptTimeout      time.Duration `json:"-"`
	AgentAcceptTimeoutSec   int           `json:"agent_accept_timeout_seconds"`
	AutoAssignBotOnTimeout  bool          `json:"auto_assign_bot_on_timeout"`
	NoAgentAvailableMessage string        `json:"no_agent_available_message"`
}

var (
	mu     sync.RWMutex
	active = defaultConfig()
)

func defaultConfig() SupportConfig {
	return SupportConfig{
		BotEnabled:              true,
		BotName:                 "Support Bot",
		BotWelcomeMessage:       "Hi! I'm here to help while we connect you with a team member.",
		AgentAcceptTimeout:      15 * time.Second,
		AgentAcceptTimeoutSec:   15,
		AutoAssignBotOnTimeout:  true,
		NoAgentAvailableMessage: "All agents are busy. Our bot assistant will help you for now.",
	}
}

func Get() SupportConfig {
	mu.RLock()
	defer mu.RUnlock()
	return active
}

func Update(cfg SupportConfig) SupportConfig {
	mu.Lock()
	defer mu.Unlock()

	if cfg.AgentAcceptTimeoutSec <= 0 {
		cfg.AgentAcceptTimeoutSec = 15
	}
	cfg.AgentAcceptTimeout = time.Duration(cfg.AgentAcceptTimeoutSec) * time.Second

	if cfg.BotName == "" {
		cfg.BotName = "Support Bot"
	}
	if cfg.BotWelcomeMessage == "" {
		cfg.BotWelcomeMessage = defaultConfig().BotWelcomeMessage
	}
	if cfg.NoAgentAvailableMessage == "" {
		cfg.NoAgentAvailableMessage = defaultConfig().NoAgentAvailableMessage
	}

	active = cfg
	return active
}
