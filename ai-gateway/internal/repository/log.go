package repository

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"ai-gateway/internal/model"
)

type LogRepo struct{}

func NewLogRepo() *LogRepo {
	return &LogRepo{}
}

func (r *LogRepo) Create(log *model.CallLog) error {
	_, err := DB.Exec(
		`INSERT INTO call_logs (token_name, channel_name, model_name, latency_ms, token_used, status, error) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		log.TokenName, log.ChannelName, log.ModelName, log.LatencyMs, log.TokenUsed, log.Status, log.Error,
	)
	return err
}

func (r *LogRepo) List(page, pageSize int, startTime, endTime *time.Time) ([]*model.CallLog, int64, error) {
	offset := (page - 1) * pageSize

	query := `SELECT COUNT(*) FROM call_logs WHERE 1=1`
	args := []interface{}{}

	if startTime != nil {
		query += ` AND created_at >= ?`
		args = append(args, startTime)
	}
	if endTime != nil {
		query += ` AND created_at <= ?`
		args = append(args, endTime)
	}

	var total int64
	if err := DB.QueryRow(query, args...).Scan(&total); err != nil {
		total = 0
	}

	selectQuery := `SELECT id, token_name, channel_name, model_name, latency_ms, token_used, status, error, created_at FROM call_logs WHERE 1=1`
	if startTime != nil {
		selectQuery += ` AND created_at >= ?`
	}
	if endTime != nil {
		selectQuery += ` AND created_at <= ?`
	}
	selectQuery += ` ORDER BY id DESC LIMIT ? OFFSET ?`

	args = append(args, pageSize, offset)

	rows, err := DB.Query(selectQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var logs []*model.CallLog
	for rows.Next() {
		l := &model.CallLog{}
		var errorMsg sql.NullString
		if err := rows.Scan(&l.ID, &l.TokenName, &l.ChannelName, &l.ModelName, &l.LatencyMs, &l.TokenUsed, &l.Status, &errorMsg, &l.CreatedAt); err != nil {
			continue
		}
		if errorMsg.Valid {
			l.Error = errorMsg.String
		}
		logs = append(logs, l)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

func (r *LogRepo) CleanupOlderThan(days int) error {
	cutoff := time.Now().AddDate(0, 0, -days)
	_, err := DB.Exec(`DELETE FROM call_logs WHERE created_at < ?`, cutoff)
	return err
}

func (r *LogRepo) GetStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})
	errorCount := 0

	var totalCalls int64
	if err := DB.QueryRow(`SELECT COUNT(*) FROM call_logs`).Scan(&totalCalls); err != nil {
		totalCalls = 0
		errorCount++
	}
	stats["total_calls"] = totalCalls

	var totalTokens int64
	if err := DB.QueryRow(`SELECT COALESCE(SUM(token_used), 0) FROM call_logs`).Scan(&totalTokens); err != nil {
		totalTokens = 0
		errorCount++
	}
	stats["total_tokens"] = totalTokens

	var avgLatency float64
	if err := DB.QueryRow(`SELECT COALESCE(AVG(latency_ms), 0) FROM call_logs`).Scan(&avgLatency); err != nil {
		avgLatency = 0
		errorCount++
	}
	stats["avg_latency"] = avgLatency

	now := time.Now()
	today := now.Format("2006-01-02")

	var todayCalls int64
	if err := DB.QueryRow(`SELECT COUNT(*) FROM call_logs WHERE date(created_at) = ?`, today).Scan(&todayCalls); err != nil {
		todayCalls = 0
		errorCount++
	}
	stats["today_calls"] = todayCalls

	var todayTokens int64
	if err := DB.QueryRow(`SELECT COALESCE(SUM(token_used), 0) FROM call_logs WHERE date(created_at) = ?`, today).Scan(&todayTokens); err != nil {
		todayTokens = 0
		errorCount++
	}
	stats["today_tokens"] = todayTokens

	weekStart := now.AddDate(0, 0, -int(now.Weekday()))
	weekStr := weekStart.Format("2006-01-02")
	var weekCalls int64
	if err := DB.QueryRow(`SELECT COUNT(*) FROM call_logs WHERE date(created_at) >= ?`, weekStr).Scan(&weekCalls); err != nil {
		weekCalls = 0
		errorCount++
	}
	stats["week_calls"] = weekCalls

	var weekTokens int64
	if err := DB.QueryRow(`SELECT COALESCE(SUM(token_used), 0) FROM call_logs WHERE date(created_at) >= ?`, weekStr).Scan(&weekTokens); err != nil {
		weekTokens = 0
		errorCount++
	}
	stats["week_tokens"] = weekTokens

	monthStart := now.Format("2006-01")
	var monthCalls int64
	if err := DB.QueryRow(`SELECT COUNT(*) FROM call_logs WHERE strftime('%Y-%m', created_at) = ?`, monthStart).Scan(&monthCalls); err != nil {
		monthCalls = 0
		errorCount++
	}
	stats["month_calls"] = monthCalls

	var monthTokens int64
	if err := DB.QueryRow(`SELECT COALESCE(SUM(token_used), 0) FROM call_logs WHERE strftime('%Y-%m', created_at) = ?`, monthStart).Scan(&monthTokens); err != nil {
		monthTokens = 0
		errorCount++
	}
	stats["month_tokens"] = monthTokens

	if errorCount > 0 && errorCount == 10 {
		return stats, fmt.Errorf("failed to retrieve %d stat queries", errorCount)
	}

	return stats, nil
}

func (r *LogRepo) GetTopChannels(limit int) ([]map[string]interface{}, error) {
	rows, err := DB.Query(`
		SELECT channel_name, COUNT(*) as call_count, AVG(latency_ms) as avg_latency 
		FROM call_logs 
		WHERE channel_name IS NOT NULL 
		GROUP BY channel_name 
		ORDER BY call_count DESC 
		LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var channelName string
		var callCount int64
		var avgLatency float64
		if err := rows.Scan(&channelName, &callCount, &avgLatency); err != nil {
			continue
		}
		results = append(results, map[string]interface{}{
			"channel_name": channelName,
			"call_count":   callCount,
			"avg_latency":  avgLatency,
		})
	}

	return results, nil
}

func (r *LogRepo) GetTokenStats(limit int) ([]map[string]interface{}, error) {
	now := time.Now()
	today := now.Format("2006-01-02")
	weekStart := now.AddDate(0, 0, -int(now.Weekday())).Format("2006-01-02")
	monthStart := now.Format("2006-01")

	rows, err := DB.Query(`
		SELECT 
			token_name,
			COUNT(*) as total_calls,
			SUM(CASE WHEN date(created_at) = ? THEN 1 ELSE 0 END) as today_calls,
			SUM(CASE WHEN date(created_at) >= ? THEN 1 ELSE 0 END) as week_calls,
			SUM(CASE WHEN strftime('%Y-%m', created_at) = ? THEN 1 ELSE 0 END) as month_calls,
			COALESCE(AVG(latency_ms), 0) as avg_latency
		FROM call_logs 
		WHERE token_name IS NOT NULL AND token_name != ''
		GROUP BY token_name 
		ORDER BY total_calls DESC 
		LIMIT ?`, today, weekStart, monthStart, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var tokenName string
		var totalCalls, todayCalls, weekCalls, monthCalls int64
		var avgLatency float64
		if err := rows.Scan(&tokenName, &totalCalls, &todayCalls, &weekCalls, &monthCalls, &avgLatency); err != nil {
			continue
		}
		results = append(results, map[string]interface{}{
			"token_name":  tokenName,
			"total_calls": totalCalls,
			"today_calls": todayCalls,
			"week_calls":  weekCalls,
			"month_calls": monthCalls,
			"avg_latency": avgLatency,
		})
	}

	return results, nil
}

func (r *LogRepo) Save(log *model.CallLog) error {
	return r.Create(log)
}

func (r *LogRepo) GetTotalStats() (*model.PageResult, error) {
	stats, err := r.GetStats()
	if err != nil {
		return nil, err
	}

	topChannels, err := r.GetTopChannels(5)
	if err != nil {
		return nil, err
	}

	tokenStats, err := r.GetTokenStats(10)
	if err != nil {
		return nil, err
	}

	return &model.PageResult{
		Total: 0,
		Items: map[string]interface{}{
			"stats":        stats,
			"top_channels": topChannels,
			"token_stats":  tokenStats,
		},
	}, nil
}

func (r *LogRepo) GetModelStatsByChannelAndModel(channelName, modelName string) (*ModelStatsResult, error) {
	result := &ModelStatsResult{}
	var totalCalls, successCalls int64
	var avgLatency float64

	err := DB.QueryRow(`
		SELECT 
			COUNT(*) as total_calls,
			SUM(CASE WHEN status < 400 THEN 1 ELSE 0 END) as success_calls,
			COALESCE(AVG(latency_ms), 0) as avg_latency
		FROM call_logs
		WHERE channel_name = ? AND model_name = ?
	`, channelName, modelName).Scan(&totalCalls, &successCalls, &avgLatency)

	if err != nil {
		return nil, err
	}

	result.TotalCalls = totalCalls
	result.SuccessCalls = successCalls
	result.AvgLatency = avgLatency

	return result, nil
}

type ModelStatsResult struct {
	TotalCalls   int64
	SuccessCalls int64
	AvgLatency   float64
}

func (r *LogRepo) GetModelStats() ([]map[string]interface{}, error) {
	rows, err := DB.Query(`
		SELECT 
			c.name as channel_name,
			m.name as model_name,
			COALESCE(stats.total_calls, 0) as total_calls,
			COALESCE(stats.success_calls, 0) as success_calls,
			COALESCE(stats.avg_latency, 0) as avg_latency,
			COALESCE(stats.avg_success_latency, 0) as avg_success_latency,
			COALESCE(stats.total_tokens, 0) as total_tokens,
			c.format,
			m.type
		FROM models m
		INNER JOIN channels c ON m.channel_id = c.id
		LEFT JOIN (
			SELECT 
				channel_name,
				model_name,
				COUNT(*) as total_calls,
				SUM(CASE WHEN status < 400 THEN 1 ELSE 0 END) as success_calls,
				AVG(latency_ms) as avg_latency,
				AVG(CASE WHEN status < 400 THEN latency_ms ELSE NULL END) as avg_success_latency,
				SUM(token_used) as total_tokens
			FROM call_logs
			WHERE channel_name IS NOT NULL AND model_name IS NOT NULL
			GROUP BY channel_name, model_name
		) stats ON c.name = stats.channel_name AND m.name = stats.model_name
		WHERE m.enabled = 1
		ORDER BY total_calls DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var channelName, modelName, format, modelType sql.NullString
		var totalCalls, successCalls int64
		var avgLatency, avgSuccessLatency, totalTokens float64
		if err := rows.Scan(&channelName, &modelName, &totalCalls, &successCalls, &avgLatency, &avgSuccessLatency, &totalTokens, &format, &modelType); err != nil {
			continue
		}

		successRate := 0.0
		if totalCalls > 0 {
			successRate = float64(successCalls) / float64(totalCalls) * 100
		}

		formatStr := "unknown"
		if format.Valid {
			formatStr = format.String
		}
		typeStr := "chat"
		if modelType.Valid && modelType.String != "" {
			typeStr = modelType.String
		}

		results = append(results, map[string]interface{}{
			"channel_name":        channelName.String,
			"model_name":          modelName.String,
			"format":              formatStr,
			"type":                typeStr,
			"total_calls":         totalCalls,
			"success_calls":       successCalls,
			"failed_calls":        totalCalls - successCalls,
			"success_rate":        successRate,
			"avg_latency":         avgLatency,
			"avg_success_latency": avgSuccessLatency,
			"total_tokens":        int64(totalTokens),
		})
	}

	return results, nil
}

func (r *LogRepo) GetModelStatsMap() map[string]*ModelStatsResult {
	result := make(map[string]*ModelStatsResult)

	rows, err := DB.Query(`
		SELECT 
			channel_name,
			model_name,
			COUNT(*) as total_calls,
			SUM(CASE WHEN status < 400 THEN 1 ELSE 0 END) as success_calls,
			COALESCE(AVG(latency_ms), 0) as avg_latency
		FROM call_logs
		GROUP BY channel_name, model_name
	`)
	if err != nil {
		return result
	}
	defer rows.Close()

	for rows.Next() {
		var channelName, modelName string
		var totalCalls, successCalls int64
		var avgLatency float64
		if err := rows.Scan(&channelName, &modelName, &totalCalls, &successCalls, &avgLatency); err != nil {
			continue
		}
		key := strings.ToLower(channelName + "::" + modelName)
		result[key] = &ModelStatsResult{
			TotalCalls:   totalCalls,
			SuccessCalls: successCalls,
			AvgLatency:   avgLatency,
		}
	}

	return result
}
