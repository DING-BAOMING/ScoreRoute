package model

import "time"

type Channel struct {
	ID              int64      `json:"id"`
	Name            string     `json:"name"`
	Format          string     `json:"format"`
	BaseURL         string     `json:"base_url"`
	APIKey          string     `json:"-"`                  // Never expose in JSON
	MaskedAPIKey    string     `json:"api_key"`           // Masked version for display
	Enabled         int        `json:"enabled"`
	CallCount       int        `json:"call_count"`
	RateLimits      string     `json:"rate_limits"`       // JSON array of rate limit rules
	TotalTokenLimit int64      `json:"total_token_limit"` // 0 = unlimited
	ExpiresAt       *time.Time `json:"expires_at"`        // nil = never expires
	TotalCalls      int64      `json:"total_calls"`       // total accumulated calls
	TotalTokens     int64      `json:"total_tokens"`      // total accumulated tokens
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type RateLimitRule struct {
	Type        string    `json:"type"`         // "calls", "tokens", or "billing"
	Window      string    `json:"window"`       // "minute", "hour", "day", "week", "month", "quarter", "year"
	MaxCount    int64     `json:"max_count"`    // maximum allowed in window (for billing, this is the max cost)
	Currency    string    `json:"currency"`     // "CNY" or "USD" (for billing type)
	CurrentCount int64    `json:"current_count"` // current count in window
	WindowStart time.Time `json:"window_start"`  // when the current window started
}

type ChannelRequest struct {
	Name            string     `json:"name" binding:"required"`
	Format          string     `json:"format" binding:"required"`
	BaseURL         string     `json:"base_url" binding:"required"`
	APIKey          string     `json:"api_key" binding:"required"`
	Enabled         int        `json:"enabled"`
	RateLimits      string     `json:"rate_limits"`
	TotalTokenLimit int64      `json:"total_token_limit"`
	ExpiresAt       *time.Time `json:"expires_at"`
}

type Model struct {
	ID              int64      `json:"id"`
	ChannelID       int64      `json:"channel_id"`
	Name            string     `json:"name"`
	Type            string     `json:"type"` // chat, embedding, etc.
	Enabled         int        `json:"enabled"`
	CallCount      int        `json:"call_count"`
	RateLimits     string     `json:"rate_limits"`      // JSON array of rate limit rules
	TotalTokenLimit int64     `json:"total_token_limit"` // 0 = unlimited
	ExpiresAt       *time.Time `json:"expires_at"`      // nil = never expires
	TotalCalls     int64      `json:"total_calls"`      // total accumulated calls
	TotalTokens    int64      `json:"total_tokens"`     // total accumulated tokens
	CostPerToken   float64    `json:"cost_per_token"`   // cost per token (in base currency)
	Currency       string     `json:"currency"`         // "CNY" or "USD"
	CreatedAt      time.Time  `json:"created_at"`
	ChannelName    string     `json:"channel_name,omitempty"`
	Format         string     `json:"format,omitempty"`
	ChannelRateLimits string   `json:"channel_rate_limits,omitempty"` // channel's rate limits for billing detection
}

type Token struct {
	ID        int64     `json:"id"`
	Key       string    `json:"-"`           // Never expose in JSON
	MaskedKey string    `json:"key"`       // Masked version for display
	Name      string    `json:"name"`
	Format    string    `json:"format"`
	Type      string    `json:"type"`
	ModelName string    `json:"model_name"`
	Enabled   int       `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
}

type SystemConfig struct {
	ID           int64     `json:"id"`
	ExchangeRate float64   `json:"exchange_rate"` // CNY to USD rate
	Currency     string    `json:"currency"`      // base currency for cost calculation
	UpdatedAt    time.Time `json:"updated_at"`
}

type CallLog struct {
	ID           int64     `json:"id"`
	TokenName    string    `json:"token_name"`
	ChannelName  string    `json:"channel_name"`
	ModelName    string    `json:"model_name"`
	LatencyMs    int       `json:"latency_ms"`
	TokenUsed    int       `json:"token_used"`
	Status       int       `json:"status"`
	Error        string    `json:"error,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// Request/Response DTOs
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

type ModelRequest struct {
	ChannelID       int64      `json:"channel_id" binding:"required"`
	Name           string     `json:"name" binding:"required"`
	Type           string     `json:"type"` // chat, embedding, etc.
	Enabled        int        `json:"enabled"`
	RateLimits     string     `json:"rate_limits"`
	TotalTokenLimit int64     `json:"total_token_limit"`
	ExpiresAt      *time.Time `json:"expires_at"`
	CostPerToken   float64    `json:"cost_per_token"`
	Currency       string     `json:"currency"`
}

type TokenRequest struct {
	Name      string `json:"name" binding:"required"`
	Format    string `json:"format" binding:"required"`
	Type      string `json:"type" binding:"required"`
	ModelName string `json:"model_name"`
	Enabled   int    `json:"enabled"`
}

type APIResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type PageResult struct {
	Total int64       `json:"total"`
	Items interface{} `json:"items"`
}

type UserRating struct {
	ID         int64     `json:"id"`
	ModelName  string    `json:"model_name"`
	UserRating int       `json:"user_rating"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type UserRatingRequest struct {
	ModelName  string `json:"model_name" binding:"required"`
	UserRating int    `json:"user_rating" binding:"required,min=1,max=100"`
}

type Sample struct {
	ID              int64     `json:"id"`
	ModelKey        string    `json:"model_key"`
	RequestContent string    `json:"request_content"`
	ResponseContent string   `json:"response_content"`
	TokenCount     int       `json:"token_count"`
	CreatedAt      time.Time `json:"created_at"`
	ExpiresAt      time.Time `json:"expires_at"`
	RemainingDays  int       `json:"remaining_days,omitempty"`
	RemainingMinutes int     `json:"remaining_minutes,omitempty"`
}

type SampleRequest struct {
	ModelKey        string `json:"model_key" binding:"required"`
	RequestContent string `json:"request_content" binding:"required"`
	ResponseContent string `json:"response_content" binding:"required"`
	TokenCount     int    `json:"token_count" binding:"required"`
}

type SampleAnalysisConfig struct {
	ID        int64     `json:"id"`
	Format    string    `json:"format"`
	BaseURL   string    `json:"base_url"`
	APIKey    string    `json:"api_key"`
	ModelName string    `json:"model_name"`
	Enabled   int       `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type SampleAnalysisLog struct {
	ID            int64     `json:"id"`
	ModelKey      string    `json:"model_key"`
	AnalysisTime  time.Time `json:"analysis_time"`
	DeleteTime    time.Time `json:"delete_time,omitempty"`
	Success       int       `json:"success"`
	ErrorMessage  string    `json:"error_message,omitempty"`
	Score         int       `json:"score"`
	AnalysisDetails string   `json:"analysis_details,omitempty"`
}

type SampleRating struct {
	ID                       int64     `json:"id"`
	ModelKey                 string    `json:"model_key"`
	Score                    int       `json:"score"`
	ToolCallingScore         int       `json:"tool_calling_score"`
	CompletenessScore        int       `json:"completeness_score"`
	ContextUnderstandingScore int       `json:"context_understanding_score"`
	ErrorHandlingScore       int       `json:"error_handling_score"`
	ResponseQualityScore     int       `json:"response_quality_score"`
	AnalyzedAt               time.Time `json:"analyzed_at"`
	ExpiresAt                time.Time `json:"expires_at"`
}

type SampleAnalysisConfigRequest struct {
	Format    string `json:"format" binding:"required"`
	BaseURL   string `json:"base_url" binding:"required"`
	APIKey    string `json:"api_key" binding:"required"`
	ModelName string `json:"model_name" binding:"required"`
	Enabled   int    `json:"enabled"`
}

type SampleRatingRequest struct {
	ModelKey string `json:"model_key" binding:"required"`
	Score    int    `json:"score" binding:"required,min=1,max=100"`
}

type ExtraRatingConfig struct {
	ID           int64      `json:"id"`
	ConfigKey    string     `json:"config_key"`
	ConfigValue string     `json:"config_value"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type ExtraRatingRecord struct {
	ID             int64      `json:"id"`
	ModelKey       string     `json:"model_key"`
	RecordType    string     `json:"record_type"`    // "penalty" or "reward"
	PenaltyScore  int        `json:"penalty_score"`  // original penalty score
	RewardScore   int        `json:"reward_score"`   // original reward score
	CurrentScore  int        `json:"current_score"` // current score after decay
	DecayPerReq   int        `json:"decay_per_request"`
	RequestCount  int        `json:"request_count"`
	CreatedAt     time.Time  `json:"created_at"`
	ExpiresAt     *time.Time `json:"expires_at"`
}

type ExtraRatingRequest struct {
	ConfigKey    string `json:"config_key" binding:"required"`
	ConfigValue  string `json:"config_value" binding:"required"`
}

type ExtraRatingResponse struct {
	PenaltyRecords []ExtraRatingRecord `json:"penalty_records"`
	RewardRecords   []ExtraRatingRecord `json:"reward_records"`
	TotalPenalty   int                `json:"total_penalty"`
	TotalReward    int                `json:"total_reward"`
}
