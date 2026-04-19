package service

import (
	"ai-gateway/internal/model"
	"fmt"
	"strings"
)

func (s *SampleAnalysisService) buildAnalysisPromptWithStrategy(sample *model.Sample, strategy ExtractionStrategy) string {
	info := extractSampleInfoWithStrategy(sample.RequestContent, sample.ResponseContent, strategy)

	var requestToolsStr string
	if len(info.RequestTools) > 0 {
		requestToolsStr = "Available tools: " + strings.Join(info.RequestTools, ", ")
	} else {
		requestToolsStr = "No tools available in request"
	}

	var responseToolsStr string
	if len(info.ResponseToolCalls) > 0 {
		responseToolsStr = strings.Join(info.ResponseToolCalls, ", ")
	} else {
		responseToolsStr = "No tools called"
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
		return s.buildMinimalPrompt(info, requestToolsStr, responseToolsStr, errorStr)
	case StrategyTailFirst:
		return s.buildReducedPrompt(info, requestToolsStr, responseToolsStr, errorStr)
	default:
		return s.buildFullPrompt(info, requestToolsStr, responseToolsStr, errorStr)
	}
}

func (s *SampleAnalysisService) buildFullPrompt(info *ExtractedSampleInfo, requestToolsStr, responseToolsStr, errorStr string) string {
	prompt := fmt.Sprintf(`Analyze this AI model response and rate it from 0-100.

## Step 1: Task Analysis
User Task: %s
System Prompt: %s

Does this task require tool calling to complete? Consider:
- Does the user ask for information that requires web search, file operations, or external APIs?
- Does the task require executing code, calculations, or data processing?
- Is the task asking to perform actions rather than just provide information?

## Step 2: Tool Usage Check
%s

Response tool calls: %s

## Error Status
%s

Please analyze and provide JSON response with:
- needs_tool_calling (boolean - does this task require tool calling)
- score (0-100 overall score)
- tool_calling_score (0-100 - was tool calling correctly handled)
- completeness_score (0-100)
- context_understanding_score (0-100)
- error_handling_score (0-100)
- response_quality_score (0-100)
- reasoning (brief explanation)

Scoring rules:
- If needs_tool_calling=true but response has no tool calls: tool_calling_score should be LOW (task incomplete)
- If needs_tool_calling=false and no tool calls: tool_calling_score should be HIGH (correct)
- If needs_tool_calling=true and has tool calls: tool_calling_score should be HIGH (correct)

Example response format:
{"needs_tool_calling": false, "score": 85, "tool_calling_score": 90, "completeness_score": 80, "context_understanding_score": 85, "error_handling_score": 85, "response_quality_score": 85, "reasoning": "Simple question answered directly without tools needed."}`, info.UserTask, info.SystemPrompt, requestToolsStr, responseToolsStr, errorStr)

	return prompt
}

func (s *SampleAnalysisService) buildReducedPrompt(info *ExtractedSampleInfo, requestToolsStr, responseToolsStr, errorStr string) string {
	prompt := fmt.Sprintf(`Analyze this AI response:

Task: %s
Available tools: %s
Response tool calls: %s
Error: %s

Does task need tool calling? Rate 0-100: score, tool_calling_score (low if needs tools but none called), completeness_score, context_understanding_score, error_handling_score, response_quality_score.
Provide needs_tool_calling boolean and reasoning.`, info.UserTask, requestToolsStr, responseToolsStr, errorStr)

	return prompt
}

func (s *SampleAnalysisService) buildMinimalPrompt(info *ExtractedSampleInfo, requestToolsStr, responseToolsStr, errorStr string) string {
	prompt := fmt.Sprintf(`Rate response 0-100:
Task: %s
Tools: %s
Response tools: %s
Error: %s

JSON with needs_tool_calling bool: {"needs_tool_calling": X, "score": X, "tool_calling_score": X, "completeness_score": X, "context_understanding_score": X, "error_handling_score": X, "response_quality_score": X, "reasoning": "..."}`, info.UserTask, requestToolsStr, responseToolsStr, errorStr)

	return prompt
}

func (s *SampleAnalysisService) buildChunkedPrompt(info *ExtractedSampleInfo, requestToolsStr, responseToolsStr, errorStr string, partIndex, totalParts int) string {
	prompt := fmt.Sprintf(`Analyze part %d/%d of a model response:

Task: %s
Available tools: %s
Response tools: %s
Error: %s

Does this part need tool calling? Rate 0-100: score, tool_calling_score, completeness_score, context_understanding_score, error_handaking_score, response_quality_score.
Provide needs_tool_calling boolean and reasoning.`, partIndex, totalParts, info.UserTask, requestToolsStr, responseToolsStr, errorStr)

	return prompt
}
