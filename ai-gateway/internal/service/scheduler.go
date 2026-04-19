package service

import (
	"log"
	"time"
)

func (d *Dispatcher) StartAutoEnableScheduler() {
	ticker := time.NewTicker(1 * time.Minute)
	go func() {
		for range ticker.C {
			d.processAutoReEnable()
		}
	}()
	log.Println("[Scheduler] Auto-enable scheduler started (interval: 1 minute)")
}

func (d *Dispatcher) processAutoReEnable() {
	log.Println("[Scheduler] Processing auto-re-enable check...")

	tokenCount := d.processAutoReEnableTokens()
	modelCount := d.processAutoReEnableModels()
	channelCount := d.processAutoReEnableChannels()

	if tokenCount > 0 || modelCount > 0 || channelCount > 0 {
		log.Printf("[Scheduler] Re-enabled: %d tokens, %d models, %d channels", tokenCount, modelCount, channelCount)
	}
}

func (d *Dispatcher) processAutoReEnableTokens() int {
	tokens, err := d.tokenRepo.GetAutoDisabledTokens()
	if err != nil {
		log.Printf("[Scheduler] Failed to get auto-disabled tokens: %v", err)
		return 0
	}

	count := 0
	for _, token := range tokens {
		if d.shouldReEnableToken(token) {
			if err := d.tokenRepo.ClearAutoDisabled(token.ID); err != nil {
				log.Printf("[Scheduler] Failed to re-enable token %d: %v", token.ID, err)
			} else {
				log.Printf("[Scheduler] Re-enabled token: %s (id=%d)", token.Name, token.ID)
				count++
			}
		}
	}

	return count
}

func (d *Dispatcher) processAutoReEnableModels() int {
	models, err := d.modelRepo.GetAutoDisabledModels()
	if err != nil {
		log.Printf("[Scheduler] Failed to get auto-disabled models: %v", err)
		return 0
	}

	count := 0
	for _, model := range models {
		if d.shouldReEnableModel(model) {
			if err := d.modelRepo.ClearAutoDisabled(model.ID); err != nil {
				log.Printf("[Scheduler] Failed to re-enable model %d: %v", model.ID, err)
			} else {
				log.Printf("[Scheduler] Re-enabled model: %s (id=%d)", model.Name, model.ID)
				count++
			}
		}
	}

	return count
}

func (d *Dispatcher) processAutoReEnableChannels() int {
	channels, err := d.channelRepo.GetAutoDisabledChannels()
	if err != nil {
		log.Printf("[Scheduler] Failed to get auto-disabled channels: %v", err)
		return 0
	}

	count := 0
	for _, channel := range channels {
		if d.shouldReEnableChannel(channel) {
			if err := d.channelRepo.ClearAutoDisabled(channel.ID); err != nil {
				log.Printf("[Scheduler] Failed to re-enable channel %d: %v", channel.ID, err)
			} else {
				log.Printf("[Scheduler] Re-enabled channel: %s (id=%d)", channel.Name, channel.ID)
				count++
			}
		}
	}

	return count
}
