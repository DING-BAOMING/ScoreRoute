package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"ai-gateway/internal/model"
)

func (d *Dispatcher) logCall(token *model.Token, channel *model.Channel, modelItem *model.Model, startTime time.Time, tokenUsed int, status int, errMsg string) {
	latency := int(time.Since(startTime).Milliseconds())

	callLog := &model.CallLog{
		TokenName:   token.Name,
		ChannelName: channel.Name,
		ModelName:   modelItem.Name,
		LatencyMs:   latency,
		TokenUsed:   tokenUsed,
		Status:      status,
		Error:       errMsg,
		CreatedAt:   time.Now(),
	}

	if err := d.logRepo.Save(callLog); err != nil {
		log.Printf("failed to save call log: %v", err)
	}
}

func (d *Dispatcher) saveSampleAsync(modelKey, requestContent, responseContent string, tokenCount int) {
	if err := d.sampleRepo.SaveSample(modelKey, requestContent, responseContent, tokenCount); err != nil {
		log.Printf("failed to save sample: %v", err)
	}
}

func (d *Dispatcher) saveSampleAsyncContext(ctx context.Context, modelKey, requestContent, responseContent string, tokenCount int) error {
	errCh := make(chan error, 1)
	go func() {
		d.saveSampleAsync(modelKey, requestContent, responseContent, tokenCount)
		errCh <- nil
	}()
	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (d *Dispatcher) updateChannelUsage(channelID int64, tokenUsed int, costPerToken float64, costCurrency string) error {
	if err := d.channelRepo.IncrementUsage(channelID, tokenUsed); err != nil {
		log.Printf("failed to increment channel usage: %v", err)
	}

	channel, err := d.channelService.GetByID(channelID)
	if err != nil || channel == nil {
		return err
	}

	if channel.RateLimits == "" || channel.RateLimits == "[]" {
		return nil
	}

	var rules []model.RateLimitRule
	if err := json.Unmarshal([]byte(channel.RateLimits), &rules); err != nil {
		log.Printf("failed to parse rate limits for channel %s: %v", channel.Name, err)
		return nil
	}

	now := time.Now()
	for idx, rule := range rules {
		windowDuration := getWindowDuration(rule.Window)
		if windowDuration == 0 {
			continue
		}

		usage, err := d.rateLimitRepo.GetUsage(channelID, idx)
		if err != nil {
			log.Printf("failed to get rate limit usage for channel %s rule %d: %v", channel.Name, idx, err)
			continue
		}

		var windowStart time.Time
		var increment int64 = 1

		if rule.Type == "tokens" {
			increment = int64(tokenUsed)
		} else if rule.Type == "billing" {
			increment = d.calculateCostInTargetCurrency(tokenUsed, costPerToken, costCurrency, rule.Currency)
		}

		if usage == nil {
			windowStart = now
			if err := d.rateLimitRepo.UpsertUsage(channelID, idx, increment, windowStart, true); err != nil {
				log.Printf("failed to upsert rate limit usage for channel %s rule %d: %v", channel.Name, idx, err)
			}
		} else {
			windowStart = usage.WindowStart
			if now.Sub(windowStart) >= windowDuration {
				windowStart = now
				increment = int64(tokenUsed)
				if rule.Type == "calls" {
					increment = 1
				} else if rule.Type == "billing" {
					increment = d.calculateCostInTargetCurrency(tokenUsed, costPerToken, costCurrency, rule.Currency)
				}
				if err := d.rateLimitRepo.UpsertUsage(channelID, idx, increment, windowStart, true); err != nil {
					log.Printf("failed to upsert rate limit usage for channel %s rule %d: %v", channel.Name, idx, err)
				}
			} else {
				if err := d.rateLimitRepo.UpsertUsage(channelID, idx, increment, windowStart, false); err != nil {
					log.Printf("failed to upsert rate limit usage for channel %s rule %d: %v", channel.Name, idx, err)
				}
			}
		}
	}

	return nil
}

func (d *Dispatcher) updateChannelUsageContext(ctx context.Context, channelID int64, tokenUsed int, costPerToken float64, costCurrency string) error {
	errCh := make(chan error, 1)
	go func() {
		errCh <- d.updateChannelUsage(channelID, tokenUsed, costPerToken, costCurrency)
	}()
	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (d *Dispatcher) checkModelRateLimit(modelItem *model.Model, tokenUsed int) error {
	if modelItem == nil {
		return nil
	}

	now := time.Now()

	if modelItem.ExpiresAt != nil && now.After(*modelItem.ExpiresAt) {
		return fmt.Errorf("model expired at %s", modelItem.ExpiresAt.Format("2006-01-02 15:04:05"))
	}

	if modelItem.TotalTokenLimit > 0 && modelItem.TotalTokens >= modelItem.TotalTokenLimit {
		return fmt.Errorf("model total token limit exceeded: %d/%d", modelItem.TotalTokens, modelItem.TotalTokenLimit)
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
			return fmt.Errorf("model calls rate limit exceeded: %d/%d per %s", currentCount, rule.MaxCount, rule.Window)
		}

		if rule.Type == "tokens" && currentCount >= rule.MaxCount {
			return fmt.Errorf("model tokens rate limit exceeded: %d/%d per %s", currentCount, rule.MaxCount, rule.Window)
		}
	}

	return nil
}

func (d *Dispatcher) checkInheritedChannelLimits(modelItem *model.Model, now time.Time) error {
	channel, err := d.channelService.GetByID(modelItem.ChannelID)
	if err != nil || channel == nil {
		return nil
	}

	if channel.RateLimits == "" || channel.RateLimits == "[]" {
		return nil
	}

	var rules []model.RateLimitRule
	if err := json.Unmarshal([]byte(channel.RateLimits), &rules); err != nil {
		log.Printf("failed to parse inherited rate limits from channel %s for model %s: %v", channel.Name, modelItem.Name, err)
		return nil
	}

	for idx, rule := range rules {
		windowDuration := getWindowDuration(rule.Window)
		if windowDuration == 0 {
			continue
		}

		usage, err := d.rateLimitRepo.GetUsage(channel.ID, idx)
		if err != nil {
			log.Printf("failed to get inherited rate limit usage from channel %s rule %d for model %s: %v", channel.Name, idx, modelItem.Name, err)
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
			return fmt.Errorf("inherited channel calls rate limit exceeded: %d/%d per %s", currentCount, rule.MaxCount, rule.Window)
		}

		if rule.Type == "tokens" && currentCount >= rule.MaxCount {
			return fmt.Errorf("inherited channel tokens rate limit exceeded: %d/%d per %s", currentCount, rule.MaxCount, rule.Window)
		}

		if rule.Type == "billing" && currentCount >= rule.MaxCount {
			return fmt.Errorf("inherited channel billing limit exceeded: %d/%s per %s (quota exhausted)", currentCount/100, rule.Currency, rule.Window)
		}
	}

	return nil
}

func (d *Dispatcher) updateModelUsage(modelID int64, tokenUsed int) error {
	if err := d.modelRepo.IncrementUsage(modelID, tokenUsed); err != nil {
		log.Printf("failed to increment model usage: %v", err)
	}

	modelItem, err := d.modelRepo.GetByID(modelID)
	if err != nil || modelItem == nil {
		return err
	}

	hasModelRules := modelItem.RateLimits != "" && modelItem.RateLimits != "[]"

	if !hasModelRules {
		d.updateInheritedChannelUsage(modelItem, tokenUsed)
		return nil
	}

	var rules []model.RateLimitRule
	if err := json.Unmarshal([]byte(modelItem.RateLimits), &rules); err != nil {
		log.Printf("failed to parse rate limits for model %s: %v", modelItem.Name, err)
		return nil
	}

	now := time.Now()
	for idx, rule := range rules {
		windowDuration := getWindowDuration(rule.Window)
		if windowDuration == 0 {
			continue
		}

		usage, err := d.modelRateLimitRepo.GetUsage(modelID, idx)
		if err != nil {
			log.Printf("failed to get rate limit usage for model %s rule %d: %v", modelItem.Name, idx, err)
			continue
		}

		var windowStart time.Time
		var increment int64 = 1

		if rule.Type == "tokens" {
			increment = int64(tokenUsed)
		}

		if usage == nil {
			windowStart = now
			if err := d.modelRateLimitRepo.UpsertUsage(modelID, idx, increment, windowStart, true); err != nil {
				log.Printf("failed to upsert rate limit usage for model %s rule %d: %v", modelItem.Name, idx, err)
			}
		} else {
			windowStart = usage.WindowStart
			if now.Sub(windowStart) >= windowDuration {
				windowStart = now
				increment = int64(tokenUsed)
				if rule.Type == "calls" {
					increment = 1
				}
				if err := d.modelRateLimitRepo.UpsertUsage(modelID, idx, increment, windowStart, true); err != nil {
					log.Printf("failed to upsert rate limit usage for model %s rule %d: %v", modelItem.Name, idx, err)
				}
			} else {
				if err := d.modelRateLimitRepo.UpsertUsage(modelID, idx, increment, windowStart, false); err != nil {
					log.Printf("failed to upsert rate limit usage for model %s rule %d: %v", modelItem.Name, idx, err)
				}
			}
		}
	}

	return nil
}

func (d *Dispatcher) updateModelUsageContext(ctx context.Context, modelID int64, tokenUsed int) error {
	errCh := make(chan error, 1)
	go func() {
		errCh <- d.updateModelUsage(modelID, tokenUsed)
	}()
	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (d *Dispatcher) updateInheritedChannelUsage(modelItem *model.Model, tokenUsed int) {
	channel, err := d.channelService.GetByID(modelItem.ChannelID)
	if err != nil || channel == nil {
		return
	}

	if channel.RateLimits == "" || channel.RateLimits == "[]" {
		return
	}

	var rules []model.RateLimitRule
	if err := json.Unmarshal([]byte(channel.RateLimits), &rules); err != nil {
		log.Printf("failed to parse inherited rate limits from channel %s for model %s: %v", channel.Name, modelItem.Name, err)
		return
	}

	now := time.Now()
	for idx, rule := range rules {
		windowDuration := getWindowDuration(rule.Window)
		if windowDuration == 0 {
			continue
		}

		usage, err := d.rateLimitRepo.GetUsage(channel.ID, idx)
		if err != nil {
			log.Printf("failed to get inherited rate limit usage from channel %s rule %d for model %s: %v", channel.Name, idx, modelItem.Name, err)
			continue
		}

		var windowStart time.Time
		var increment int64 = 1

		if rule.Type == "tokens" {
			increment = int64(tokenUsed)
		} else if rule.Type == "billing" {
			increment = d.calculateCostInTargetCurrency(tokenUsed, modelItem.CostPerToken, modelItem.Currency, rule.Currency)
		}

		if usage == nil {
			windowStart = now
			if err := d.rateLimitRepo.UpsertUsage(channel.ID, idx, increment, windowStart, true); err != nil {
				log.Printf("failed to upsert inherited rate limit usage for channel %s rule %d: %v", channel.Name, idx, err)
			}
		} else {
			windowStart = usage.WindowStart
			if now.Sub(windowStart) >= windowDuration {
				windowStart = now
				increment = int64(tokenUsed)
				if rule.Type == "calls" {
					increment = 1
				} else if rule.Type == "billing" {
					increment = d.calculateCostInTargetCurrency(tokenUsed, modelItem.CostPerToken, modelItem.Currency, rule.Currency)
				}
				if err := d.rateLimitRepo.UpsertUsage(channel.ID, idx, increment, windowStart, true); err != nil {
					log.Printf("failed to upsert inherited rate limit usage for channel %s rule %d: %v", channel.Name, idx, err)
				}
			} else {
				if err := d.rateLimitRepo.UpsertUsage(channel.ID, idx, increment, windowStart, false); err != nil {
					log.Printf("failed to upsert inherited rate limit usage for channel %s rule %d: %v", channel.Name, idx, err)
				}
			}
		}
	}
}
