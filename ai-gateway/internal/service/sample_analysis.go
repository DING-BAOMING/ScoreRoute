package service

import (
	"ai-gateway/internal/model"
	"ai-gateway/internal/repository"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"
)

type SampleAnalysisService struct {
	configRepo *repository.SampleAnalysisConfigRepo
	logRepo    *repository.SampleAnalysisLogRepo
	ratingRepo *repository.SampleRatingRepo
	sampleRepo *repository.SampleRepo
	client     *http.Client
}

func NewSampleAnalysisService() *SampleAnalysisService {
	return &SampleAnalysisService{
		configRepo: repository.NewSampleAnalysisConfigRepo(),
		logRepo:    repository.NewSampleAnalysisLogRepo(),
		ratingRepo: repository.NewSampleRatingRepo(),
		sampleRepo: repository.NewSampleRepo(),
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

type AnalysisPrompt struct {
	ModelKey         string `json:"model_key"`
	RequestContent  string `json:"request_content"`
	ResponseContent string `json:"response_content"`
}

type AnalysisResult struct {
	Score                    int    `json:"score"`
	ToolCallingScore         int    `json:"tool_calling_score"`
	CompletenessScore        int    `json:"completeness_score"`
	ContextUnderstandingScore int    `json:"context_understanding_score"`
	ErrorHandlingScore       int    `json:"error_handling_score"`
	ResponseQualityScore     int    `json:"response_quality_score"`
	Reasoning                string `json:"reasoning"`
}

func (s *SampleAnalysisService) GetConfig() (*model.SampleAnalysisConfig, error) {
	return s.configRepo.Get()
}

func (s *SampleAnalysisService) SaveConfig(req *model.SampleAnalysisConfigRequest) error {
	return s.configRepo.Upsert(req)
}

func (s *SampleAnalysisService) TestConnection(req *model.SampleAnalysisConfigRequest) (bool, string, error) {
	testPrompt := fmt.Sprintf(`{"model":"%s","messages":[{"role":"user","content":"Say 'test successful' in exactly those words"}],"max_tokens":20}`, req.ModelName)

	httpReq, err := http.NewRequest("POST", req.BaseURL+"/chat/completions", strings.NewReader(testPrompt))
	if err != nil {
		return false, "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+req.APIKey)

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return false, fmt.Sprintf("connection failed: %v", err), nil
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return false, fmt.Sprintf("API returned status %d: %s", resp.StatusCode, string(body)), nil
	}

	return true, "Connection successful", nil
}

func (s *SampleAnalysisService) AnalyzeSample(sample *model.Sample) (*AnalysisResult, error) {
	config, err := s.configRepo.GetEnabled()
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}
	if config == nil {
		return nil, fmt.Errorf("sample analysis not configured")
	}

	prompt := s.buildAnalysisPrompt(sample)

	requestBody := map[string]interface{}{
		"model": config.ModelName,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"max_tokens": 500,
	}

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", config.BaseURL+"/chat/completions", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+config.APIKey)

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	return s.parseAnalysisResponse(body)
}

func (s *SampleAnalysisService) buildAnalysisPrompt(sample *model.Sample) string {
	requestContent, _ := json.Marshal(sample.RequestContent)
	responseContent, _ := json.Marshal(sample.ResponseContent)
	
	const maxRequestLen = 3000
	const maxResponseLen = 1500
	truncatedRequest := string(requestContent)
	truncatedResponse := string(responseContent)
	if len(truncatedRequest) > maxRequestLen {
		truncatedRequest = truncatedRequest[:maxRequestLen] + "...[truncated]"
	}
	if len(truncatedResponse) > maxResponseLen {
		truncatedResponse = truncatedResponse[:maxResponseLen] + "...[truncated]"
	}
	
	prompt := fmt.Sprintf(`You are an AI agent evaluation expert. Analyze the following sample and rate the model's performance for agentic tasks.

## Evaluation Criteria (1-100 scale each):

1. **Tool Calling Score (30%% weight)**: Does the model correctly identify when to call tools? Does it call the right tools with correct parameters? Look for patterns like function_call objects, explicit tool mentions, etc.

2. **Completeness Score (25%% weight)**: Does the response fully address the user's request? Are all parts addressed? Is it complete?

3. **Context Understanding (20%% weight)**: Does the model understand the context? Does it maintain conversation coherence?

4. **Error Handling (15%% weight)**: How does it handle ambiguous or missing information? Does it ask for clarification when needed?

5. **Response Quality (10%% weight)**: General response quality - clarity and formatting.

## Sample to Analyze:

**Model**: %s
**Request**: %s
**Response**: %s

## Output Format:
Return ONLY a valid JSON object with this exact structure, no markdown or additional text:
{"score":85,"tool_calling_score":90,"completeness_score":85,"context_understanding_score":80,"error_handling_score":75,"response_quality_score":90,"reasoning":"Brief explanation"}

Analyze and return ONLY the JSON.`, sample.ModelKey, truncatedRequest, truncatedResponse)

	return prompt
}

func (s *SampleAnalysisService) parseAnalysisResponse(body []byte) (*AnalysisResult, error) {
	var resp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	content := resp.Choices[0].Message.Content
	content = strings.TrimSpace(content)

	var result AnalysisResult
	
	if err := json.Unmarshal([]byte(content), &result); err == nil && result.Score > 0 {
		return &result, nil
	}

	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	if err := json.Unmarshal([]byte(content), &result); err == nil {
		s.setDefaultScoresIfNeeded(&result)
		return &result, nil
	}

	jsonStart := strings.Index(content, "{")
	jsonEnd := strings.LastIndex(content, "}")
	if jsonStart != -1 && jsonEnd != -1 && jsonEnd > jsonStart {
		jsonStr := content[jsonStart : jsonEnd+1]
		if err := json.Unmarshal([]byte(jsonStr), &result); err == nil {
			s.setDefaultScoresIfNeeded(&result)
			return &result, nil
		}
	}

	contentPreview := content
	if len(contentPreview) > 200 {
		contentPreview = contentPreview[:200] + "..."
	}
	return nil, fmt.Errorf("failed to parse analysis result as JSON. Response preview: %s", contentPreview)
}

func (s *SampleAnalysisService) setDefaultScoresIfNeeded(result *AnalysisResult) {
	if result.Score == 0 {
		result.Score = 50
	}
	if result.ToolCallingScore == 0 {
		result.ToolCallingScore = 50
	}
	if result.CompletenessScore == 0 {
		result.CompletenessScore = 50
	}
	if result.ContextUnderstandingScore == 0 {
		result.ContextUnderstandingScore = 50
	}
	if result.ErrorHandlingScore == 0 {
		result.ErrorHandlingScore = 50
	}
	if result.ResponseQualityScore == 0 {
		result.ResponseQualityScore = 50
	}
}

func (s *SampleAnalysisService) RunScheduledAnalysis(maxSamples int) (int, error) {
	config, err := s.configRepo.GetEnabled()
	if err != nil || config == nil {
		log.Printf("Sample analysis skipped: not configured or error: %v", err)
		return 0, nil
	}

	samples, err := s.sampleRepo.List()
	if err != nil {
		return 0, err
	}

	if len(samples) == 0 {
		return 0, nil
	}

	sort.Slice(samples, func(i, j int) bool {
		return samples[i].ExpiresAt.Before(samples[j].ExpiresAt)
	})

	if len(samples) > maxSamples {
		samples = samples[:maxSamples]
	}

	analyzed := 0
	for _, sample := range samples {
		result, err := s.AnalyzeSampleWithRetry(sample, 3)
		analysisLog := &model.SampleAnalysisLog{
			ModelKey:     sample.ModelKey,
			AnalysisTime: time.Now(),
			Success:      0,
		}

		if err != nil {
			analysisLog.ErrorMessage = fmt.Sprintf("failed after 3 retries: %v", err)
			analysisLog.Score = 0
			log.Printf("Sample %s analysis failed after 3 retries: %v", sample.ModelKey, err)
		} else {
			analysisLog.Success = 1
			analysisLog.Score = result.Score
			details, _ := json.Marshal(result)
			analysisLog.AnalysisDetails = string(details)

			rating := &model.SampleRating{
				ModelKey:                 sample.ModelKey,
				Score:                    result.Score,
				ToolCallingScore:         result.ToolCallingScore,
				CompletenessScore:        result.CompletenessScore,
				ContextUnderstandingScore: result.ContextUnderstandingScore,
				ErrorHandlingScore:       result.ErrorHandlingScore,
				ResponseQualityScore:     result.ResponseQualityScore,
			}
			if err := s.ratingRepo.Upsert(rating); err != nil {
				log.Printf("Failed to save rating for %s: %v", sample.ModelKey, err)
			}

			if err := s.sampleRepo.Delete(sample.ID); err != nil {
				log.Printf("Failed to delete sample %d: %v", sample.ID, err)
			} else {
				analysisLog.DeleteTime = time.Now()
			}
			analyzed++
		}

		if err := s.logRepo.Create(analysisLog); err != nil {
			log.Printf("Failed to create analysis log: %v", err)
		}
	}

	return analyzed, nil
}

func (s *SampleAnalysisService) AnalyzeSampleWithRetry(sample *model.Sample, maxRetries int) (*AnalysisResult, error) {
	var lastErr error
	
	for retry := 0; retry < maxRetries; retry++ {
		if retry > 0 {
			backoffDuration := time.Duration(retry*2) * time.Second
			log.Printf("Retrying sample %s (attempt %d/%d) after %v", sample.ModelKey, retry+1, maxRetries, backoffDuration)
			time.Sleep(backoffDuration)
		}
		
		result, err := s.AnalyzeSample(sample)
		if err == nil {
			return result, nil
		}
		lastErr = err
		
		if retry < maxRetries-1 {
			log.Printf("Sample %s analysis attempt %d/%d failed: %v", sample.ModelKey, retry+1, maxRetries, err)
		}
	}
	
	return nil, fmt.Errorf("all retries exhausted: %w", lastErr)
}

func (s *SampleAnalysisService) GetLogs(limit int) ([]*model.SampleAnalysisLog, error) {
	if limit <= 0 {
		limit = 100
	}
	return s.logRepo.List(limit)
}

func (s *SampleAnalysisService) GetLogStats() (map[string]interface{}, error) {
	return s.logRepo.GetStats()
}

func (s *SampleAnalysisService) GetRatings() ([]*model.SampleRating, error) {
	return s.ratingRepo.List()
}

func (s *SampleAnalysisService) GetRatingsMap() (map[string]*model.SampleRating, error) {
	return s.ratingRepo.GetAllAsMap()
}

func (s *SampleAnalysisService) UpdateRating(modelKey string, score int) error {
	rating, err := s.ratingRepo.GetByModelKey(modelKey)
	if err != nil {
		return err
	}
	if rating == nil {
		rating = &model.SampleRating{
			ModelKey: modelKey,
			Score:    score,
		}
	} else {
		rating.Score = score
	}
	return s.ratingRepo.Upsert(rating)
}

func (s *SampleAnalysisService) CleanupExpiredRatings() error {
	deleted, err := s.ratingRepo.DeleteExpired()
	if err != nil {
		return err
	}
	if deleted > 0 {
		log.Printf("Cleaned up %d expired sample ratings", deleted)
	}
	return nil
}
