package service

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"ai-gateway/internal/model"
	"ai-gateway/internal/repository"
)

const (
	ReasonTokenRateLimit   = "token_rate_limit_exceeded"
	ReasonModelRateLimit   = "model_rate_limit_exceeded"
	ReasonTokenTotalLimit  = "token_total_token_limit_exceeded"
	ReasonChannelRateLimit = "channel_rate_limit_exceeded"
)

func (d *Dispatcher) disableTokenAndRelatedModels(token *model.Token, reason string) error {
	log.Printf("[AutoDisable] Disabling token %s (id=%d), reason: %s", token.Name, token.ID, reason)

	if err := d.tokenRepo.SetAutoDisabled(token.ID, reason); err != nil {
		log.Printf("[AutoDisable] Failed to disable token %d: %v", token.ID, err)
		return err
	}

	modelName := strings.ToLower(token.ModelName)

	if modelName != "__auto__" && modelName != "__poll_all__" && modelName != "*" && modelName != "" {
		channels, err := d.channelService.GetByFormatAndType(token.Format, token.Type)
		if err != nil || len(channels) == 0 {
			log.Printf("[AutoDisable] No channels found for format=%s, type=%s", token.Format, token.Type)
			return nil
		}

		for _, channel := range channels {
			model, err := d.modelRepo.GetByChannelAndName(channel.ID, token.ModelName)
			if err != nil || model == nil {
				continue
			}

			if err := d.modelRepo.SetAutoDisabled(model.ID, reason); err != nil {
				log.Printf("[AutoDisable] Failed to disable model %s (id=%d): %v", model.Name, model.ID, err)
			} else {
				log.Printf("[AutoDisable] Disabled model: %s/%s", channel.Name, model.Name)
			}
		}
	} else {
		log.Printf("[AutoDisable] Token uses auto-selection, only disabling token (not models)")
	}

	return nil
}

func (d *Dispatcher) disableModel(modelItem *model.Model, reason string) error {
	log.Printf("[AutoDisable] Disabling model %s (id=%d), reason: %s", modelItem.Name, modelItem.ID, reason)

	if err := d.modelRepo.SetAutoDisabled(modelItem.ID, reason); err != nil {
		log.Printf("[AutoDisable] Failed to disable model %d: %v", modelItem.ID, err)
		return err
	}

	return nil
}

func (d *Dispatcher) checkAndDisableTokenRateLimit(token *model.Token) error {
	now := time.Now()

	if token.ExpiresAt != nil && now.After(*token.ExpiresAt) {
		return fmt.Errorf("token expired at %s", token.ExpiresAt.Format("2006-01-02 15:04:05"))
	}

	if token.TotalTokenLimit > 0 && token.TotalTokens >= token.TotalTokenLimit {
		d.disableTokenAndRelatedModels(token, ReasonTokenTotalLimit)
		return fmt.Errorf("token total token limit exceeded: %d/%d - permanently disabled", token.TotalTokens, token.TotalTokenLimit)
	}

	if token.AutoDisabled == 1 {
		return fmt.Errorf("token is auto-disabled: %s", token.AutoDisableReason)
	}

	if token.RateLimits == "" || token.RateLimits == "[]" {
		return nil
	}

	var rules []model.RateLimitRule
	if err := json.Unmarshal([]byte(token.RateLimits), &rules); err != nil {
		log.Printf("failed to parse rate limits for token %s: %v", token.Name, err)
		return nil
	}

	for idx, rule := range rules {
		windowDuration := getWindowDuration(rule.Window)
		if windowDuration == 0 {
			continue
		}

		usage, err := d.tokenRateLimitRepo.GetUsage(token.ID, idx)
		if err != nil {
			log.Printf("failed to get rate limit usage for token %s rule %d: %v", token.Name, idx, err)
			continue
		}

		var currentCount int64
		var windowStart time.Time

		if usage == nil {
			currentCount = 0
			windowStart = now
		} else {
			currentCount = usage.CurrentCount
			windowStart = usage.WindowStart

			if now.Sub(windowStart) >= windowDuration {
				currentCount = 0
				windowStart = now
				d.tokenRateLimitRepo.UpsertUsage(token.ID, idx, 0, windowStart, true)
			}
		}

		if rule.Type == "calls" && currentCount >= rule.MaxCount {
			d.disableTokenAndRelatedModels(token, ReasonTokenRateLimit)
			return fmt.Errorf("token calls rate limit exceeded: %d/%d per %s - auto-disabled", currentCount, rule.MaxCount, rule.Window)
		}

		if rule.Type == "tokens" && currentCount >= rule.MaxCount {
			d.disableTokenAndRelatedModels(token, ReasonTokenRateLimit)
			return fmt.Errorf("token tokens rate limit exceeded: %d/%d per %s - auto-disabled", currentCount, rule.MaxCount, rule.Window)
		}
	}

	return nil
}

func (d *Dispatcher) checkAndDisableModelRateLimit(modelItem *model.Model, tokenUsed int) error {
	if modelItem == nil {
		return nil
	}

	now := time.Now()

	if modelItem.ExpiresAt != nil && now.After(*modelItem.ExpiresAt) {
		return fmt.Errorf("model expired at %s", modelItem.ExpiresAt.Format("2006-01-02 15:04:05"))
	}

	if modelItem.TotalTokenLimit > 0 && modelItem.TotalTokens >= modelItem.TotalTokenLimit {
		d.disableModel(modelItem, ReasonModelRateLimit)
		return fmt.Errorf("model total token limit exceeded: %d/%d - auto-disabled", modelItem.TotalTokens, modelItem.TotalTokenLimit)
	}

	if modelItem.AutoDisabled == 1 {
		return fmt.Errorf("model is auto-disabled: %s", modelItem.AutoDisableReason)
	}

	hasModelRules := modelItem.RateLimits != "" && modelItem.RateLimits != "[]"

	if !hasModelRules {
		return d.checkInheritedChannelLimits(modelItem, now)
	}

	var rules []model.RateLimitRule
	if err := json.Unmarshal([]byte(modelItem.RateLimits), &rules); err != nil {
		log.Printf("failed to parse rate limits for model %s: %v", modelItem.Name, err)
		return nil
	}

	for idx, rule := range rules {
		windowDuration := getWindowDuration(rule.Window)
		if windowDuration == 0 {
			continue
		}

		usage, err := d.modelRateLimitRepo.GetUsage(modelItem.ID, idx)
		if err != nil {
			log.Printf("failed to get rate limit usage for model %s rule %d: %v", modelItem.Name, idx, err)
			continue
		}

		var currentCount int64
		var windowStart time.Time

		if usage == nil {
			currentCount = 0
			windowStart = now
		} else {
			currentCount = usage.CurrentCount
			windowStart = usage.WindowStart

			if now.Sub(windowStart) >= windowDuration {
				currentCount = 0
				windowStart = now
				d.modelRateLimitRepo.UpsertUsage(modelItem.ID, idx, 0, windowStart, true)
			}
		}

		if rule.Type == "calls" && currentCount >= rule.MaxCount {
			d.disableModel(modelItem, ReasonModelRateLimit)
			return fmt.Errorf("model calls rate limit exceeded: %d/%d per %s - auto-disabled", currentCount, rule.MaxCount, rule.Window)
		}

		if rule.Type == "tokens" && currentCount >= rule.MaxCount {
			d.disableModel(modelItem, ReasonModelRateLimit)
			return fmt.Errorf("model tokens rate limit exceeded: %d/%d per %s - auto-disabled", currentCount, rule.MaxCount, rule.Window)
		}
	}

	return nil
}

func (d *Dispatcher) checkAndDisableChannelRateLimit(channel *model.Channel, tokenUsed int) error {
	now := time.Now()

	if channel.ExpiresAt != nil && now.After(*channel.ExpiresAt) {
		return fmt.Errorf("channel expired at %s", channel.ExpiresAt.Format("2006-01-02 15:04:05"))
	}

	if channel.TotalTokenLimit > 0 && channel.TotalTokens >= channel.TotalTokenLimit {
		d.channelRepo.SetAutoDisabled(channel.ID, ReasonChannelRateLimit)
		return fmt.Errorf("channel total token limit exceeded: %d/%d - auto-disabled", channel.TotalTokens, channel.TotalTokenLimit)
	}

	if channel.AutoDisabled == 1 {
		return fmt.Errorf("channel is auto-disabled: %s", channel.AutoDisableReason)
	}

	if channel.RateLimits == "" || channel.RateLimits == "[]" {
		return nil
	}

	var rules []model.RateLimitRule
	if err := json.Unmarshal([]byte(channel.RateLimits), &rules); err != nil {
		log.Printf("failed to parse rate limits for channel %s: %v", channel.Name, err)
		return nil
	}

	for idx, rule := range rules {
		windowDuration := getWindowDuration(rule.Window)
		if windowDuration == 0 {
			continue
		}

		usage, err := d.rateLimitRepo.GetUsage(channel.ID, idx)
		if err != nil {
			log.Printf("failed to get rate limit usage for channel %s rule %d: %v", channel.Name, idx, err)
			continue
		}

		var currentCount int64
		var windowStart time.Time

		if usage == nil {
			currentCount = 0
			windowStart = now
		} else {
			currentCount = usage.CurrentCount
			windowStart = usage.WindowStart

			if now.Sub(windowStart) >= windowDuration {
				currentCount = 0
				windowStart = now
				d.rateLimitRepo.UpsertUsage(channel.ID, idx, 0, windowStart, true)
			}
		}

		if rule.Type == "calls" && currentCount >= rule.MaxCount {
			d.channelRepo.SetAutoDisabled(channel.ID, ReasonChannelRateLimit)
			return fmt.Errorf("channel calls rate limit exceeded: %d/%d per %s - auto-disabled", currentCount, rule.MaxCount, rule.Window)
		}

		if rule.Type == "tokens" && currentCount >= rule.MaxCount {
			d.channelRepo.SetAutoDisabled(channel.ID, ReasonChannelRateLimit)
			return fmt.Errorf("channel tokens rate limit exceeded: %d/%d per %s - auto-disabled", currentCount, rule.MaxCount, rule.Window)
		}

		if rule.Type == "billing" && currentCount >= rule.MaxCount {
			d.channelRepo.SetAutoDisabled(channel.ID, ReasonChannelRateLimit)
			return fmt.Errorf("channel billing limit exceeded: %d/%s per %s - auto-disabled", currentCount/100, rule.Currency, rule.Window)
		}
	}

	return nil
}

func (d *Dispatcher) shouldReEnableToken(token *model.Token) bool {
	if token.AutoDisabled != 1 {
		return false
	}

	if token.AutoDisableReason == ReasonTokenTotalLimit {
		return false
	}

	if token.RateLimits == "" || token.RateLimits == "[]" {
		return false
	}

	var rules []model.RateLimitRule
	if err := json.Unmarshal([]byte(token.RateLimits), &rules); err != nil {
		return false
	}

	now := time.Now()

	for idx, rule := range rules {
		windowDuration := getWindowDuration(rule.Window)
		if windowDuration == 0 {
			continue
		}

		usage, err := d.tokenRateLimitRepo.GetUsage(token.ID, idx)
		if err != nil || usage == nil {
			return true
		}

		if now.Sub(usage.WindowStart) >= windowDuration {
			return true
		}
	}

	return false
}

func (d *Dispatcher) shouldReEnableModel(modelItem *model.Model) bool {
	if modelItem.AutoDisabled != 1 {
		return false
	}

	if modelItem.AutoDisableReason == ReasonModelRateLimit {
		return false
	}

	hasModelRules := modelItem.RateLimits != "" && modelItem.RateLimits != "[]"

	var rules []model.RateLimitRule

	if hasModelRules {
		if err := json.Unmarshal([]byte(modelItem.RateLimits), &rules); err != nil {
			return false
		}
	} else {
		channel, chErr := d.channelService.GetByID(modelItem.ChannelID)
		if chErr != nil || channel == nil {
			return false
		}

		if channel.RateLimits == "" || channel.RateLimits == "[]" {
			return false
		}

		if err := json.Unmarshal([]byte(channel.RateLimits), &rules); err != nil {
			return false
		}
	}

	now := time.Now()

	for idx, rule := range rules {
		windowDuration := getWindowDuration(rule.Window)
		if windowDuration == 0 {
			continue
		}

		var usage *repository.ModelRateLimitUsage
		var err error

		if hasModelRules {
			usage, err = d.modelRateLimitRepo.GetUsage(modelItem.ID, idx)
		} else {
			usage, err = d.modelRateLimitRepo.GetUsage(modelItem.ID, idx)
		}

		if err != nil || usage == nil {
			return true
		}

		if now.Sub(usage.WindowStart) >= windowDuration {
			return true
		}
	}

	return false
}

func (d *Dispatcher) shouldReEnableChannel(channel *model.Channel) bool {
	if channel.AutoDisabled != 1 {
		return false
	}

	if channel.RateLimits == "" || channel.RateLimits == "[]" {
		return false
	}

	var rules []model.RateLimitRule
	if err := json.Unmarshal([]byte(channel.RateLimits), &rules); err != nil {
		return false
	}

	now := time.Now()

	for idx, rule := range rules {
		windowDuration := getWindowDuration(rule.Window)
		if windowDuration == 0 {
			continue
		}

		usage, err := d.rateLimitRepo.GetUsage(channel.ID, idx)
		if err != nil || usage == nil {
			return true
		}

		if now.Sub(usage.WindowStart) >= windowDuration {
			return true
		}
	}

	return false
}
