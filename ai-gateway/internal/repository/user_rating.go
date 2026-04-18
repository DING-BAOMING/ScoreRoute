package repository

import (
	"ai-gateway/internal/model"
	"strings"
)

type UserRatingRepo struct{}

func NewUserRatingRepo() *UserRatingRepo {
	return &UserRatingRepo{}
}

func (r *UserRatingRepo) Upsert(modelName string, rating int) error {
	normalizedName := r.NormalizeModelName(modelName)
	if normalizedName == "" {
		normalizedName = strings.ToLower(strings.TrimSpace(modelName))
	}
	normalizedName = strings.ToLower(normalizedName)
	_, err := DB.Exec(`
		INSERT INTO user_ratings (model_name, user_rating, updated_at)
		VALUES (?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(model_name) DO UPDATE SET user_rating = ?, updated_at = CURRENT_TIMESTAMP`,
		normalizedName, rating, rating)
	return err
}

func (r *UserRatingRepo) GetByName(modelName string) (*model.UserRating, error) {
	normalizedName := strings.ToLower(strings.TrimSpace(modelName))
	row := DB.QueryRow(`SELECT id, model_name, user_rating, created_at, updated_at FROM user_ratings WHERE model_name = ?`, normalizedName)

	var ur model.UserRating
	err := row.Scan(&ur.ID, &ur.ModelName, &ur.UserRating, &ur.CreatedAt, &ur.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &ur, nil
}

func (r *UserRatingRepo) List() ([]*model.UserRating, error) {
	rows, err := DB.Query(`SELECT id, model_name, user_rating, created_at, updated_at FROM user_ratings ORDER BY model_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ratings []*model.UserRating
	for rows.Next() {
		var ur model.UserRating
		if err := rows.Scan(&ur.ID, &ur.ModelName, &ur.UserRating, &ur.CreatedAt, &ur.UpdatedAt); err != nil {
			continue
		}
		ratings = append(ratings, &ur)
	}
	return ratings, nil
}

func (r *UserRatingRepo) Delete(id int64) error {
	_, err := DB.Exec(`DELETE FROM user_ratings WHERE id = ?`, id)
	return err
}

func (r *UserRatingRepo) GetAllAsMap() (map[string]int, error) {
	rows, err := DB.Query(`SELECT model_name, user_rating FROM user_ratings`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]int)
	for rows.Next() {
		var modelName string
		var rating int
		if err := rows.Scan(&modelName, &rating); err != nil {
			continue
		}
		result[modelName] = rating
	}
	return result, nil
}

func (r *UserRatingRepo) GetDeduplicatedModelNames() ([]string, error) {
	rows, err := DB.Query(`
		SELECT DISTINCT LOWER(model_name) as model_name 
		FROM models 
		WHERE model_name IS NOT NULL AND model_name != ''
		ORDER BY model_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			continue
		}
		names = append(names, name)
	}
	return names, nil
}

func (r *UserRatingRepo) GetUserRatingForModel(modelName string) (int, error) {
	normalizedName := strings.ToLower(strings.TrimSpace(modelName))
	var rating int
	err := DB.QueryRow(`SELECT user_rating FROM user_ratings WHERE model_name = ?`, normalizedName).Scan(&rating)
	if err != nil {
		return 50, err
	}
	return rating, nil
}

func (r *UserRatingRepo) GetUserRatingForNormalizedModel(modelName string) (int, error) {
	normalizedKey := r.NormalizeModelName(modelName)
	var rating int
	err := DB.QueryRow(`SELECT user_rating FROM user_ratings WHERE model_name = ?`, normalizedKey).Scan(&rating)
	if err != nil {
		return 50, nil
	}
	return rating, nil
}

func (r *UserRatingRepo) GetAllUserRatings() ([]map[string]interface{}, error) {
	rows, err := DB.Query(`
		SELECT DISTINCT LOWER(m.name) as model_name, COALESCE(u.user_rating, 50) as user_rating
		FROM models m
		LEFT JOIN user_ratings u ON LOWER(m.name) = u.model_name
		WHERE m.name IS NOT NULL AND m.name != ''
		ORDER BY m.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var modelName string
		var userRating int
		if err := rows.Scan(&modelName, &userRating); err != nil {
			continue
		}
		results = append(results, map[string]interface{}{
			"model_name":  modelName,
			"user_rating": userRating,
		})
	}
	return results, nil
}

func (r *UserRatingRepo) NormalizeModelName(name string) string {
	name = strings.TrimSpace(name)
	lowerName := strings.ToLower(name)

	if strings.HasPrefix(lowerName, "minimaxai/") {
		name = strings.ToLower(name)
		name = strings.TrimPrefix(name, "minimaxai/")
		name = strings.TrimPrefix(name, "minimax-")
		name = strings.TrimPrefix(name, "minimax")
		if len(name) > 1 && name[0] == 'm' && name[1] >= '0' && name[1] <= '9' {
			name = name[1:]
		}
		return "MiniMax-" + name
	}

	if strings.HasPrefix(lowerName, "minimax-") || strings.HasPrefix(lowerName, "minimax") {
		name = strings.ToLower(name)
		name = strings.TrimPrefix(name, "minimax-")
		name = strings.TrimPrefix(name, "minimax")
		if len(name) > 1 && name[0] == 'm' && name[1] >= '0' && name[1] <= '9' {
			name = name[1:]
		}
		return "MiniMax-" + name
	}

	name = strings.ToLower(name)

	vendorPrefixes := []string{"google/", "qwen/", "z-ai/", "anthropic/", "openai/", "meta/", "mistral/", "cohere/", "azure/", "aws/", "alibaba/", "baidu/", "tencent/"}
	for _, prefix := range vendorPrefixes {
		if strings.HasPrefix(name, prefix) {
			name = strings.TrimPrefix(name, prefix)
			break
		}
	}

	return name
}

func (r *UserRatingRepo) GetDeduplicatedUserRatings() ([]map[string]interface{}, error) {
	rows, err := DB.Query(`SELECT name FROM models WHERE name IS NOT NULL AND name != ''`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var allNames []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			continue
		}
		allNames = append(allNames, name)
	}

	deduped := make(map[string]string)
	for _, name := range allNames {
		normalized := r.NormalizeModelName(name)
		key := strings.ToLower(normalized)
		if _, exists := deduped[key]; !exists {
			deduped[key] = name
		}
	}

	ratingRows, err := DB.Query(`SELECT model_name, user_rating FROM user_ratings`)
	if err != nil {
		return nil, err
	}
	defer ratingRows.Close()

	ratingsMap := make(map[string]int)
	for ratingRows.Next() {
		var modelName string
		var rating int
		ratingRows.Scan(&modelName, &rating)
		ratingsMap[modelName] = rating
	}

	var results []map[string]interface{}
	for key, originalName := range deduped {
		rating := 50
		if r, ok := ratingsMap[key]; ok {
			rating = r
		}
		displayName := r.toDisplayName(key)
		results = append(results, map[string]interface{}{
			"model_name":    displayName,
			"original_name": originalName,
			"user_rating":   rating,
		})
	}
	return results, nil
}

func (r *UserRatingRepo) toDisplayName(name string) string {
	name = strings.ToLower(name)
	switch name {
	case "minimax-2.1", "minimax-2.5", "minimax-2.7":
		return "MiniMax-" + strings.TrimPrefix(name, "minimax-")
	case "glm5":
		return "GLM-5"
	case "glm-4.7-flash":
		return "GLM-4.7-Flash"
	case "cogvideox-flash":
		return "CogVideoX-Flash"
	case "qwen3-coder-480b-a35b-instruct":
		return "Qwen3-Coder-480b-A35b-Instruct"
	case "gemma-3-27b-it":
		return "Gemma-3-27b-It"
	}
	if strings.HasPrefix(name, "minimax-") {
		return "MiniMax-" + strings.TrimPrefix(name, "minimax-")
	}
	return name
}
