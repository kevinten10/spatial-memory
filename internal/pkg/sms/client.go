package sms

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"

	"github.com/spatial-memory/spatial-memory/internal/config"
)

// Client is the interface for sending SMS verification codes.
type Client interface {
	SendCode(ctx context.Context, phone, code string) error
}

type client struct {
	cfg config.SMSConfig
}

func NewClient(cfg config.SMSConfig) Client {
	return &client{cfg: cfg}
}

func (c *client) SendCode(ctx context.Context, phone, code string) error {
	// TODO: integrate real SMS provider (Aliyun SMS, Tencent Cloud SMS, etc.)
	// For development, just log the code.
	log.Info().
		Str("phone", phone).
		Str("code", code).
		Str("provider", c.cfg.Provider).
		Msg("SMS code sent (dev mode)")

	if c.cfg.APIKey == "" {
		return nil // Dev mode: no real sending
	}

	return fmt.Errorf("SMS provider %s not yet implemented", c.cfg.Provider)
}
