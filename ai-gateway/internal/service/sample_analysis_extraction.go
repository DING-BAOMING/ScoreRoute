package service

import (
	"encoding/json"
	"fmt"
	"strings"
)

type ExtractedSampleInfo struct {
	Model        string   `json:"model"`
	UserTask     string   `json:"user_task"`
	SystemPrompt string   `json:"system_prompt,omitempty"`
	RequestTools      []string `json:"request_tools,omitempty"`
	ResponseToolCalls []string `json:"response_tool_calls,omitempty"`
	Completion   string   `json:"completion"`
	HasError     bool     `json:"has_error"`
	ErrorMsg     string   `json:"error_msg,omitempty"`
	ResponseLen  int      `json:"response_length"`
}

type ExtractionStrategy int

const (
	StrategyHeadFirst ExtractionStrategy = iota
	StrategyTailFirst
	StrategyMinimal
)

const (
	maxSystemPromptLenHead = 400
	maxUserTaskLenHead     = 600
	maxCompletionLenHead   = 1000

	maxSystemPromptLenTail = 200
	maxUserTaskLenTail     = 300
	maxCompletionLenTail   = 500

	maxSystemPromptLenMin = 100
	maxUserTaskLenMin     = 150
	maxCompletionLenMin   = 200
)

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func truncateString(s string, maxLen int, truncationType string) string {
	if len(s) <= maxLen {
		return s
	}
	switch truncationType {
	case "head":
		return s[:maxLen] + "...[head]"
	case "tail":
		return "...[tail]" + s[len(s)-maxLen:]
	case "middle":
		if maxLen <= 10 {
			return s[:maxLen] + "..."
		}
		half := maxLen / 2
		return s[:half] + "...[middle]..." + s[len(s)-half:]
	default:
		return s[:maxLen] + "..."
	}
}

func extractSampleInfo(requestJSON, responseJSON string) *ExtractedSampleInfo {
	info := &ExtractedSampleInfo{
		Completion: "unknown",
		HasError:   false,
	}

	var req, resp map[string]interface{}
	if err := json.Unmarshal([]byte(requestJSON), &req); err != nil {
		info.UserTask = requestJSON
		if len(info.UserTask) > 500 {
			info.UserTask = info.UserTask[:500] + "...[parsed failed]"
		}
		return info
	}

	if messages, ok := req["messages"].([]interface{}); ok {
		for _, msg := range messages {
			if m, ok := msg.(map[string]interface{}); ok {
				role, _ := m["role"].(string)
				content, _ := m["content"].(string)
				if role == "system" {
					if len(content) > 300 {
						info.SystemPrompt = content[:300] + "...[truncated]"
					} else {
						info.SystemPrompt = content
					}
				} else if role == "user" && info.UserTask == "" {
					if len(content) > 500 {
						info.UserTask = content[:500] + "...[truncated]"
					} else {
						info.UserTask = content
					}
				}
			}
		}
	}

	if toolCalls, ok := req["tools"].([]interface{}); ok {
		for _, tc := range toolCalls {
			if t, ok := tc.(map[string]interface{}); ok {
				if name, ok := t["name"].(string); ok {
					info.RequestTools = append(info.RequestTools, name)
				}
			}
		}
	}

	if err := json.Unmarshal([]byte(responseJSON), &resp); err == nil {
		info.ResponseLen = len(responseJSON)

		if choices, ok := resp["choices"].([]interface{}); ok && len(choices) > 0 {
			if choice, ok := choices[0].(map[string]interface{}); ok {
				if msg, ok := choice["message"].(map[string]interface{}); ok {
					if content, ok := msg["content"].(string); ok {
						if len(content) > 800 {
							info.Completion = content[:800] + "...[truncated]"
						} else {
							info.Completion = content
						}
					}
					if tc, ok := msg["tool_calls"].([]interface{}); ok {
						for _, call := range tc {
							if c, ok := call.(map[string]interface{}); ok {
								if fn, ok := c["function"].(map[string]interface{}); ok {
									if name, ok := fn["name"].(string); ok {
										info.ResponseToolCalls = append(info.ResponseToolCalls, name)
									}
								}
							}
						}
					}
				}
			}
		}

		if errObj, ok := resp["error"].(map[string]interface{}); ok {
			info.HasError = true
			if msg, ok := errObj["message"].(string); ok {
				info.ErrorMsg = msg
			}
		}
	} else {
		info.Completion = responseJSON
		if len(info.Completion) > 800 {
			info.Completion = info.Completion[:800] + "...[truncated]"
		}
	}

	return info
}

func extractSampleInfoWithStrategy(requestJSON, responseJSON string, strategy ExtractionStrategy) *ExtractedSampleInfo {
	info := extractSampleInfo(requestJSON, responseJSON)

	var maxSystemLen, maxUserLen, maxCompletionLen int
	var systemTruncType, userTruncType, completionTruncType string

	switch strategy {
	case StrategyHeadFirst:
		maxSystemLen = maxSystemPromptLenHead
		maxUserLen = maxUserTaskLenHead
		maxCompletionLen = maxCompletionLenHead
		systemTruncType = "head"
		userTruncType = "head"
		completionTruncType = "head"
	case StrategyTailFirst:
		maxSystemLen = maxSystemPromptLenTail
		maxUserLen = maxUserTaskLenTail
		maxCompletionLen = maxCompletionLenTail
		systemTruncType = "tail"
		userTruncType = "tail"
		completionTruncType = "tail"
	case StrategyMinimal:
		maxSystemLen = maxSystemPromptLenMin
		maxUserLen = maxUserTaskLenMin
		maxCompletionLen = maxCompletionLenMin
		systemTruncType = "middle"
		userTruncType = "middle"
		completionTruncType = "middle"
	}

	if len(info.SystemPrompt) > maxSystemLen && info.SystemPrompt != "" {
		info.SystemPrompt = truncateString(info.SystemPrompt, maxSystemLen, systemTruncType)
	}
	if len(info.UserTask) > maxUserLen {
		info.UserTask = truncateString(info.UserTask, maxUserLen, userTruncType)
	}
	if len(info.Completion) > maxCompletionLen && info.Completion != "unknown" {
		info.Completion = truncateString(info.Completion, maxCompletionLen, completionTruncType)
	}

	return info
}

func extractSampleInfoFull(requestJSON, responseJSON string) *ExtractedSampleInfo {
	info := extractSampleInfo(requestJSON, responseJSON)
	return info
}

func formatTools(tools []string) string {
	if len(tools) == 0 {
		return "No tools called"
	}
	result := "Tools called: " + strings.Join(tools[:min(5, len(tools))], ", ")
	if len(tools) > 5 {
		result += fmt.Sprintf(" (+%d more)", len(tools)-5)
	}
	return result
}
