package repository

import (
	"ai-gateway/internal/model"
	"database/sql"
	"time"
)

type SampleAnalysisLogRepo struct{}

func NewSampleAnalysisLogRepo() *SampleAnalysisLogRepo {
	return &SampleAnalysisLogRepo{}
}

func (r *SampleAnalysisLogRepo) Create(log *model.SampleAnalysisLog) error {
	_, err := DB.Exec(`
		INSERT INTO sample_analysis_logs (model_key, analysis_time, delete_time, success, error_message, score, analysis_details)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, log.ModelKey, log.AnalysisTime, log.DeleteTime, log.Success, log.ErrorMessage, log.Score, log.AnalysisDetails)
	return err
}

func (r *SampleAnalysisLogRepo) List(limit int) ([]*model.SampleAnalysisLog, error) {
	rows, err := DB.Query(`
		SELECT id, model_key, analysis_time, delete_time, success, error_message, score, analysis_details
		FROM sample_analysis_logs 
		ORDER BY analysis_time DESC 
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*model.SampleAnalysisLog
	for rows.Next() {
		l := &model.SampleAnalysisLog{}
		var deleteTime sql.NullTime
		var errorMsg sql.NullString
		var analysisDetails sql.NullString
		if err := rows.Scan(&l.ID, &l.ModelKey, &l.AnalysisTime, &deleteTime, &l.Success, &errorMsg, &l.Score, &analysisDetails); err != nil {
			continue
		}
		if deleteTime.Valid {
			l.DeleteTime = deleteTime.Time
		}
		if errorMsg.Valid {
			l.ErrorMessage = errorMsg.String
		}
		if analysisDetails.Valid {
			l.AnalysisDetails = analysisDetails.String
		}
		logs = append(logs, l)
	}
	return logs, rows.Err()
}

func (r *SampleAnalysisLogRepo) UpdateDeleteTime(id int64, deleteTime time.Time) error {
	_, err := DB.Exec(`UPDATE sample_analysis_logs SET delete_time = ? WHERE id = ?`, deleteTime, id)
	return err
}

func (r *SampleAnalysisLogRepo) GetStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	var total int64
	if err := DB.QueryRow(`SELECT COUNT(*) FROM sample_analysis_logs`).Scan(&total); err != nil {
		total = 0
	}
	stats["total"] = total

	var successCount int64
	if err := DB.QueryRow(`SELECT COUNT(*) FROM sample_analysis_logs WHERE success = 1`).Scan(&successCount); err != nil {
		successCount = 0
	}
	stats["success_count"] = successCount

	var failedCount int64
	if err := DB.QueryRow(`SELECT COUNT(*) FROM sample_analysis_logs WHERE success = 0`).Scan(&failedCount); err != nil {
		failedCount = 0
	}
	stats["failed_count"] = failedCount

	var avgScore float64
	if err := DB.QueryRow(`SELECT COALESCE(AVG(score), 0) FROM sample_analysis_logs WHERE success = 1`).Scan(&avgScore); err != nil {
		avgScore = 0
	}
	stats["avg_score"] = avgScore

	var deletedCount int64
	if err := DB.QueryRow(`SELECT COUNT(*) FROM sample_analysis_logs WHERE delete_time IS NOT NULL`).Scan(&deletedCount); err != nil {
		deletedCount = 0
	}
	stats["deleted_count"] = deletedCount

	return stats, nil
}
