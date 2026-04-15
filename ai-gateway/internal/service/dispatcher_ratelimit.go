package service

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"ai-gateway/internal/model"
)

// checkRateLimit 检查渠道的速率限制
func (d *Dispatcher) checkRateLimit(channel *model.Channel, tokenUsed int) error {
	now := time.Now()

	if channel.ExpiresAt != nil && now.After(*channel.ExpiresAt) {
		return fmt.Errorf("channel expired at %s", channel.ExpiresAt.Format("2006-01-02 15:04:05"))
	}

	if channel.TotalTokenLimit > 0 && channel.TotalTokens >= channel.TotalTokenLimit {
		return fmt.Errorf("total token limit exceeded: %d/%d", channel.TotalTokens, channel.TotalTokenLimit)
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
			return fmt.Errorf("calls rate limit exceeded: %d/%d per %s", currentCount, rule.MaxCount, rule.Window)
		}

		if rule.Type == "tokens" && currentCount >= rule.MaxCount {
			return fmt.Errorf("tokens rate limit exceeded: %d/%d per %s", currentCount, rule.MaxCount, rule.Window)
		}

		if rule.Type == "billing" && currentCount >= rule.MaxCount {
			return fmt.Errorf("billing limit exceeded: %d/%s per %s (quota exhausted)", currentCount/100, rule.Currency, rule.Window)
		}
	}

	return nil
}

// getExchangeRate 获取汇率配置
func (d *Dispatcher) getExchangeRate() float64 {
	config, err := d.systemConfigRepo.Get()
	if err != nil || config == nil {
		return 7.25
	}
	return config.ExchangeRate
}

// calculateCostInTargetCurrency 计算目标货币的成本
func (d *Dispatcher) calculateCostInTargetCurrency(tokenUsed int, costPerToken float64, costCurrency string, targetCurrency string) int64 {
	if tokenUsed <= 0 || costPerToken <= 0 {
		return 0
	}
	baseCost := float64(tokenUsed) * costPerToken
	if costCurrency == targetCurrency {
		return int64(baseCost * 100)
	}
	exchangeRate := d.getExchangeRate()
	if costCurrency == "USD" && targetCurrency == "CNY" {
		return int64(baseCost * exchangeRate * 100)
	}
	if costCurrency == "CNY" && targetCurrency == "USD" {
		return int64(baseCost / exchangeRate * 100)
	}
	return int64(baseCost * 100)
}
