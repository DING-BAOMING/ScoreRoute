package service

import (
	"ai-gateway/internal/model"
	"fmt"
)

func (s *SampleAnalysisService) buildAnalysisPromptWithStrategy(sample *model.Sample, strategy ExtractionStrategy) string {
	info := extractSampleInfoWithStrategy(sample.RequestContent, sample.ResponseContent, strategy)

	var toolCallsStr string
	if len(info.ToolCalls) > 0 {
		toolCallsStr = formatTools(info.ToolCalls)
	} else {
		toolCallsStr = "No tools called"
	}

	errorStr := ""
	if info.HasError {
		errorStr = fmt.Sprintf("ERROR: %s", info.ErrorMsg)
		if len(errorStr) > 100 {
			errorStr = errorStr[:100] + "...[truncated]"
		}
	}

	switch strategy {
	case StrategyMinimal:
		return s.buildMinimalPrompt(info, toolCallsStr, errorStr)
	case StrategyTailFirst:
		return s.buildReducedPrompt(info, toolCallsStr, errorStr)
	default:
		return s.buildFullPrompt(info, toolCallsStr, errorStr)
	}
}

func (s *SampleAnalysisService) buildFullPrompt(info *ExtractedSampleInfo, toolCallsStr, errorStr string) string {
	prompt := fmt.Sprintf(`Analyze this AI model response and rate it from 0-100.

## User Task
%s

## System Prompt
%s

## Model Response
%s

## Tool Usage
%s

## Error Status
%s

Please provide a JSON response with:
- score (0-100)
- tool_calling_score (0-100)
- completeness_score (0-100)
- context_understanding_score (0-100)
- error_handling_score (0-100)
- response_quality_score (0-100)
- reasoning (brief explanation)

Example response format:
{"score": 75, "tool_calling_score": 80, "completeness_score": 70, "context_understanding_score": 85, "error_handling_score": 60, "response_quality_score": 75, "reasoning": "The response was helpful but incomplete."}`, info.UserTask, info.SystemPrompt, info.Completion, toolCallsStr, errorStr)

	return prompt
}

func (s *SampleAnalysisService) buildReducedPrompt(info *ExtractedSampleInfo, toolCallsStr, errorStr string) string {
	prompt := fmt.Sprintf(`Analyze this AI response (focus on recent parts):

## Task
%s

## Response
%s

## Tools: %s
## Error: %s

Rate 0-100: score, tool_calling_score, completeness_score, context_understanding_score, error_handling_score, response_quality_score, reasoning.`, info.UserTask, info.Completion, toolCallsStr, errorStr)

	return prompt
}

func (s *SampleAnalysisService) buildMinimalPrompt(info *ExtractedSampleInfo, toolCallsStr, errorStr string) string {
	prompt := fmt.Sprintf(`Rate this response 0-100:
Task: %s
Response: %s
Tools: %s
Error: %s

JSON: {"score": X, "tool_calling_score": X, "completeness_score": X, "context_understanding_score": X, "error_handling_score": X, "response_quality_score": X, "reasoning": "..."}`, info.UserTask, info.Completion, toolCallsStr, errorStr)

	return prompt
}

func (s *SampleAnalysisService) buildChunkedPrompt(info *ExtractedSampleInfo, partIndex, totalParts int) string {
	prompt := fmt.Sprintf(`Analyze part %d/%d of a model response:

## Task
%s

## System Prompt
%s

## Response Part %d
%s

## Tools: %s
## Error: %s

Rate this part 0-100 for: score, tool_calling_score, completeness_score, context_understanding_score, error_handling_score, response_quality_score.
Provide reasoning.`, partIndex, totalParts, info.UserTask, info.SystemPrompt, partIndex, info.Completion, formatTools(info.ToolCalls), info.ErrorMsg)

	return prompt
}
