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
	ModelKey        string `json:"model_key"`
	RequestContent  string `json:"request_content"`
	ResponseContent string `json:"response_content"`
}

type AnalysisResult struct {
	NeedsToolCalling         bool   `json:"needs_tool_calling"`
	Score                     int    `json:"score"`
	ToolCallingScore          int    `json:"tool_calling_score"`
	CompletenessScore         int    `json:"completeness_score"`
	ContextUnderstandingScore int    `json:"context_understanding_score"`
	ErrorHandlingScore        int    `json:"error_handling_score"`
	ResponseQualityScore      int    `json:"response_quality_score"`
	Reasoning                 string `json:"reasoning"`
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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Sprintf("failed to read response: %v", err), nil
	}
	if resp.StatusCode >= 400 {
		return false, fmt.Sprintf("API returned status %d: %s", resp.StatusCode, string(body)), nil
	}

	return true, "Connection successful", nil
}

func (s *SampleAnalysisService) AnalyzeSample(sample *model.Sample) (*AnalysisResult, error) {
	return s.AnalyzeSampleWithStrategy(sample, StrategyHeadFirst)
}

func (s *SampleAnalysisService) AnalyzeSampleWithStrategy(sample *model.Sample, strategy ExtractionStrategy) (*AnalysisResult, error) {
	config, err := s.configRepo.GetEnabled()
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}
	if config == nil {
		return nil, fmt.Errorf("sample analysis not configured")
	}

	prompt := s.buildAnalysisPromptWithStrategy(sample, strategy)

	maxTokens := 500
	if strategy == StrategyMinimal {
		maxTokens = 200
	} else if strategy == StrategyTailFirst {
		maxTokens = 300
	}

	return s.analyzeWithPrompt(prompt, config, maxTokens)
}

func (s *SampleAnalysisService) analyzeWithPrompt(prompt string, config *model.SampleAnalysisConfig, maxTokens int) (*AnalysisResult, error) {
	requestBody := map[string]interface{}{
		"model": config.ModelName,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"max_tokens": maxTokens,
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
	if !result.NeedsToolCalling {
		result.NeedsToolCalling = false
	}
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
		result, contextFailed, err := s.AnalyzeSampleSmartRetry(sample)
		analysisLog := &model.SampleAnalysisLog{
			ModelKey:     sample.ModelKey,
			AnalysisTime: time.Now(),
			Success:      0,
		}

		if err != nil {
			analysisLog.ErrorMessage = fmt.Sprintf("%v", err)
			analysisLog.Score = 0

			if contextFailed {
				log.Printf("Sample %s analysis failed due to context limits after all attempts, deleting sample: %v", sample.ModelKey, err)
				if err := s.sampleRepo.Delete(sample.ID); err != nil {
					log.Printf("Failed to delete sample %d: %v", sample.ID, err)
				} else {
					analysisLog.DeleteTime = time.Now()
				}
			} else {
				log.Printf("Sample %s analysis failed (non-context error), keeping sample: %v", sample.ModelKey, err)
			}
		} else {
			analysisLog.Success = 1
			analysisLog.Score = result.Score
			details, _ := json.Marshal(result)
			analysisLog.AnalysisDetails = string(details)

			rating := &model.SampleRating{
				ModelKey:                  sample.ModelKey,
				Score:                     result.Score,
				ToolCallingScore:          result.ToolCallingScore,
				CompletenessScore:         result.CompletenessScore,
				ContextUnderstandingScore: result.ContextUnderstandingScore,
				ErrorHandlingScore:        result.ErrorHandlingScore,
				ResponseQualityScore:      result.ResponseQualityScore,
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
