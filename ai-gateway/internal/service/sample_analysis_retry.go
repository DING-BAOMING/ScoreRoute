package service

import (
	"ai-gateway/internal/model"
	"strings"
)

func isContextLimitError(err error) bool {
	if err == nil {
		return false
	}
	errMsg := err.Error()
	contextLimitIndicators := []string{
		"context deadline exceeded",
		"context canceled",
		"context exceeded",
		"token limit",
		"too many tokens",
		"maximum context",
		"context length",
		"context window",
		"sequence length",
		"maximum length",
	}
	errMsgLower := strings.ToLower(errMsg)
	for _, indicator := range contextLimitIndicators {
		if strings.Contains(errMsgLower, indicator) {
			return true
		}
	}
	return false
}

func (s *SampleAnalysisService) AnalyzeSampleSmartRetry(sample *model.Sample) (*AnalysisResult, bool, error) {
	result, err := s.AnalyzeSampleWithStrategy(sample, StrategyHeadFirst)
	if err == nil {
		return result, false, nil
	}

	if !isContextLimitError(err) {
		return nil, false, err
	}

	result, err = s.AnalyzeSampleWithStrategy(sample, StrategyTailFirst)
	if err == nil {
		return result, true, nil
	}

	if !isContextLimitError(err) {
		return nil, false, err
	}

	result, err = s.AnalyzeSampleWithStrategy(sample, StrategyMinimal)
	if err == nil {
		return result, true, nil
	}

	return nil, false, err
}

func (s *SampleAnalysisService) AnalyzeSampleChunked(sample *model.Sample) (*AnalysisResult, error) {
	info := extractSampleInfo(sample.RequestContent, sample.ResponseContent)

	responseLen := len(info.Completion)
	if responseLen <= 2000 {
		return s.AnalyzeSampleWithStrategy(sample, StrategyHeadFirst)
	}

	partSize := 1500
	parts := (responseLen + partSize - 1) / partSize

	var totalScore, totalToolScore, totalCompleteScore, totalContextScore, totalErrorScore, totalQualityScore int
	validParts := 0

	for i := 0; i < parts; i++ {
		start := i * partSize
		end := start + partSize
		if end > responseLen {
			end = responseLen
		}

		partResponse := info.Completion[start:end]
		partInfo := &ExtractedSampleInfo{
			Model:        info.Model,
			UserTask:     info.UserTask,
			SystemPrompt: info.SystemPrompt,
			ToolCalls:    info.ToolCalls,
			Completion:   partResponse,
			HasError:     info.HasError,
			ErrorMsg:     info.ErrorMsg,
			ResponseLen:  len(partResponse),
		}

		prompt := s.buildChunkedPrompt(partInfo, i+1, parts)

		config, err := s.configRepo.GetEnabled()
		if err != nil || config == nil {
			continue
		}

		chunkResult, err := s.analyzeWithPrompt(prompt, config, 200)
		if err != nil {
			continue
		}

		totalScore += chunkResult.Score
		totalToolScore += chunkResult.ToolCallingScore
		totalCompleteScore += chunkResult.CompletenessScore
		totalContextScore += chunkResult.ContextUnderstandingScore
		totalErrorScore += chunkResult.ErrorHandlingScore
		totalQualityScore += chunkResult.ResponseQualityScore
		validParts++
	}

	if validParts == 0 {
		return s.AnalyzeSampleWithStrategy(sample, StrategyMinimal)
	}

	finalResult := &AnalysisResult{
		Score:                    totalScore / validParts,
		ToolCallingScore:         totalToolScore / validParts,
		CompletenessScore:        totalCompleteScore / validParts,
		ContextUnderstandingScore: totalContextScore / validParts,
		ErrorHandlingScore:       totalErrorScore / validParts,
		ResponseQualityScore:     totalQualityScore / validParts,
		Reasoning:                "Averaged from chunked analysis",
	}

	return finalResult, nil
}
