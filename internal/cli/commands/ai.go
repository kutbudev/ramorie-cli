package commands

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kutbudev/ramorie-cli/internal/api"
	apierrors "github.com/kutbudev/ramorie-cli/internal/errors"
	"github.com/urfave/cli/v2"
)

// NewAICommand creates all subcommands for the 'ai' command group.
func NewAICommand() *cli.Command {
	return &cli.Command{
		Name:    "ai",
		Usage:   "AI-powered task analysis and recommendations",
		Subcommands: []*cli.Command{
			aiNextStepCmd(),
			aiEstimateCmd(),
			aiRisksCmd(),
			aiDependenciesCmd(),
		},
	}
}

// aiNextStepCmd gets AI suggestion for the next action on a task.
func aiNextStepCmd() *cli.Command {
	return &cli.Command{
		Name:      "next-step",
		Aliases:   []string{"next"},
		Usage:     "Get AI suggested next action for a task",
		ArgsUsage: "[task-id]",
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("task ID is required")
			}
			taskID := c.Args().First()

			client := api.NewClient()

			// Get task details first for context
			task, err := client.GetTask(taskID)
			if err != nil {
				fmt.Println(apierrors.ParseAPIError(err))
				return err
			}

			fmt.Printf("🧠 Analyzing task: %s\n", truncateString(task.Title, 50))
			fmt.Println(strings.Repeat("-", 60))

			result, err := client.AINextStep(taskID)
			if err != nil {
				fmt.Println(apierrors.ParseAPIError(err))
				return err
			}

			// Format and display the AI response
			if suggestion, ok := result["suggestion"].(string); ok && suggestion != "" {
				fmt.Printf("✨ Next Step Suggestion:\n\n%s\n", suggestion)
			}

			if reasoning, ok := result["reasoning"].(string); ok && reasoning != "" {
				fmt.Printf("\n💡 Reasoning:\n%s\n", reasoning)
			}

			if steps, ok := result["steps"].([]interface{}); ok && len(steps) > 0 {
				fmt.Printf("\n📋 Recommended Steps:\n")
				for i, step := range steps {
					if stepStr, ok := step.(string); ok {
						fmt.Printf("  %d. %s\n", i+1, stepStr)
					}
				}
			}

			// If result is just raw data, pretty print it
			if len(result) > 0 && !containsKey(result, "suggestion") {
				prettyJSON, _ := json.MarshalIndent(result, "", "  ")
				fmt.Printf("\n%s\n", string(prettyJSON))
			}

			return nil
		},
	}
}

// aiEstimateCmd gets AI time estimation for a task.
func aiEstimateCmd() *cli.Command {
	return &cli.Command{
		Name:      "estimate",
		Aliases:   []string{"est"},
		Usage:     "Get AI time estimation for a task",
		ArgsUsage: "[task-id]",
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("task ID is required")
			}
			taskID := c.Args().First()

			client := api.NewClient()

			// Get task details first for context
			task, err := client.GetTask(taskID)
			if err != nil {
				fmt.Println(apierrors.ParseAPIError(err))
				return err
			}

			fmt.Printf("⏱️  Estimating task: %s\n", truncateString(task.Title, 50))
			fmt.Println(strings.Repeat("-", 60))

			result, err := client.AIEstimateTime(taskID)
			if err != nil {
				fmt.Println(apierrors.ParseAPIError(err))
				return err
			}

			// Format and display the AI response
			if estimate, ok := result["estimate"].(string); ok && estimate != "" {
				fmt.Printf("⏰ Estimated Time: %s\n", estimate)
			}

			if min, ok := result["min_hours"].(float64); ok {
				if max, ok := result["max_hours"].(float64); ok {
					fmt.Printf("📊 Range: %.1f - %.1f hours\n", min, max)
				}
			}

			if confidence, ok := result["confidence"].(string); ok && confidence != "" {
				confidenceEmoji := getConfidenceEmoji(confidence)
				fmt.Printf("%s Confidence: %s\n", confidenceEmoji, confidence)
			}

			if factors, ok := result["factors"].([]interface{}); ok && len(factors) > 0 {
				fmt.Printf("\n🔍 Factors Considered:\n")
				for _, factor := range factors {
					if factorStr, ok := factor.(string); ok {
						fmt.Printf("  • %s\n", factorStr)
					}
				}
			}

			if reasoning, ok := result["reasoning"].(string); ok && reasoning != "" {
				fmt.Printf("\n💭 Analysis:\n%s\n", reasoning)
			}

			// If result is just raw data, pretty print it
			if len(result) > 0 && !containsKey(result, "estimate") {
				prettyJSON, _ := json.MarshalIndent(result, "", "  ")
				fmt.Printf("\n%s\n", string(prettyJSON))
			}

			return nil
		},
	}
}

// aiRisksCmd analyzes potential risks for a task.
func aiRisksCmd() *cli.Command {
	return &cli.Command{
		Name:      "risks",
		Usage:     "Analyze potential risks for a task",
		ArgsUsage: "[task-id]",
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("task ID is required")
			}
			taskID := c.Args().First()

			client := api.NewClient()

			// Get task details first for context
			task, err := client.GetTask(taskID)
			if err != nil {
				fmt.Println(apierrors.ParseAPIError(err))
				return err
			}

			fmt.Printf("⚠️  Analyzing risks for: %s\n", truncateString(task.Title, 50))
			fmt.Println(strings.Repeat("-", 60))

			result, err := client.AIRisks(taskID)
			if err != nil {
				fmt.Println(apierrors.ParseAPIError(err))
				return err
			}

			// Format and display the AI response
			if riskLevel, ok := result["risk_level"].(string); ok && riskLevel != "" {
				riskEmoji := getRiskEmoji(riskLevel)
				fmt.Printf("%s Overall Risk Level: %s\n", riskEmoji, strings.ToUpper(riskLevel))
			}

			if risks, ok := result["risks"].([]interface{}); ok && len(risks) > 0 {
				fmt.Printf("\n🚨 Identified Risks:\n")
				for i, risk := range risks {
					if riskMap, ok := risk.(map[string]interface{}); ok {
						category := getStringField(riskMap, "category")
						description := getStringField(riskMap, "description")
						severity := getStringField(riskMap, "severity")

						severityEmoji := getSeverityEmoji(severity)
						fmt.Printf("\n  %d. [%s] %s\n", i+1, category, description)
						if severity != "" {
							fmt.Printf("     %s Severity: %s\n", severityEmoji, severity)
						}

						if mitigation := getStringField(riskMap, "mitigation"); mitigation != "" {
							fmt.Printf("     💡 Mitigation: %s\n", mitigation)
						}
					} else if riskStr, ok := risk.(string); ok {
						fmt.Printf("  • %s\n", riskStr)
					}
				}
			}

			if recommendations, ok := result["recommendations"].([]interface{}); ok && len(recommendations) > 0 {
				fmt.Printf("\n✅ Recommendations:\n")
				for _, rec := range recommendations {
					if recStr, ok := rec.(string); ok {
						fmt.Printf("  • %s\n", recStr)
					}
				}
			}

			if reasoning, ok := result["reasoning"].(string); ok && reasoning != "" {
				fmt.Printf("\n💭 Analysis:\n%s\n", reasoning)
			}

			// If result is just raw data, pretty print it
			if len(result) > 0 && !containsKey(result, "risks") && !containsKey(result, "risk_level") {
				prettyJSON, _ := json.MarshalIndent(result, "", "  ")
				fmt.Printf("\n%s\n", string(prettyJSON))
			}

			return nil
		},
	}
}

// aiDependenciesCmd identifies task dependencies.
func aiDependenciesCmd() *cli.Command {
	return &cli.Command{
		Name:      "dependencies",
		Aliases:   []string{"deps"},
		Usage:     "Identify dependencies for a task",
		ArgsUsage: "[task-id]",
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("task ID is required")
			}
			taskID := c.Args().First()

			client := api.NewClient()

			// Get task details first for context
			task, err := client.GetTask(taskID)
			if err != nil {
				fmt.Println(apierrors.ParseAPIError(err))
				return err
			}

			fmt.Printf("🔗 Analyzing dependencies for: %s\n", truncateString(task.Title, 50))
			fmt.Println(strings.Repeat("-", 60))

			result, err := client.AIDependencies(taskID)
			if err != nil {
				fmt.Println(apierrors.ParseAPIError(err))
				return err
			}

			// Format and display the AI response
			if deps, ok := result["dependencies"].([]interface{}); ok && len(deps) > 0 {
				fmt.Printf("📋 Identified Dependencies:\n")
				for i, dep := range deps {
					if depMap, ok := dep.(map[string]interface{}); ok {
						name := getStringField(depMap, "name")
						depType := getStringField(depMap, "type")
						status := getStringField(depMap, "status")

						typeEmoji := getDependencyTypeEmoji(depType)
						statusEmoji := getStatusEmoji(status)

						fmt.Printf("\n  %d. %s %s\n", i+1, typeEmoji, name)
						if depType != "" {
							fmt.Printf("     Type: %s\n", depType)
						}
						if status != "" {
							fmt.Printf("     %s Status: %s\n", statusEmoji, status)
						}

						if description := getStringField(depMap, "description"); description != "" {
							fmt.Printf("     📝 %s\n", description)
						}
					} else if depStr, ok := dep.(string); ok {
						fmt.Printf("  • %s\n", depStr)
					}
				}
			} else {
				fmt.Println("✨ No external dependencies identified!")
			}

			if blockers, ok := result["blockers"].([]interface{}); ok && len(blockers) > 0 {
				fmt.Printf("\n🚧 Potential Blockers:\n")
				for _, blocker := range blockers {
					if blockerStr, ok := blocker.(string); ok {
						fmt.Printf("  ⛔ %s\n", blockerStr)
					}
				}
			}

			if prerequisites, ok := result["prerequisites"].([]interface{}); ok && len(prerequisites) > 0 {
				fmt.Printf("\n📌 Prerequisites:\n")
				for _, prereq := range prerequisites {
					if prereqStr, ok := prereq.(string); ok {
						fmt.Printf("  ✓ %s\n", prereqStr)
					}
				}
			}

			if reasoning, ok := result["reasoning"].(string); ok && reasoning != "" {
				fmt.Printf("\n💭 Analysis:\n%s\n", reasoning)
			}

			// If result is just raw data, pretty print it
			if len(result) > 0 && !containsKey(result, "dependencies") {
				prettyJSON, _ := json.MarshalIndent(result, "", "  ")
				fmt.Printf("\n%s\n", string(prettyJSON))
			}

			return nil
		},
	}
}

// Helper functions

func containsKey(m map[string]interface{}, key string) bool {
	_, exists := m[key]
	return exists
}

func getStringField(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

func getConfidenceEmoji(confidence string) string {
	switch strings.ToLower(confidence) {
	case "high":
		return "🟢"
	case "medium":
		return "🟡"
	case "low":
		return "🔴"
	default:
		return "⚪"
	}
}

func getRiskEmoji(riskLevel string) string {
	switch strings.ToLower(riskLevel) {
	case "critical":
		return "🔴"
	case "high":
		return "🟠"
	case "medium":
		return "🟡"
	case "low":
		return "🟢"
	default:
		return "⚪"
	}
}

func getSeverityEmoji(severity string) string {
	switch strings.ToLower(severity) {
	case "critical":
		return "🔴"
	case "high":
		return "🟠"
	case "medium":
		return "🟡"
	case "low":
		return "🟢"
	default:
		return "⚪"
	}
}

func getDependencyTypeEmoji(depType string) string {
	switch strings.ToLower(depType) {
	case "technical":
		return "⚙️"
	case "resource":
		return "👥"
	case "data":
		return "💾"
	case "external":
		return "🌐"
	case "internal":
		return "🏢"
	default:
		return "🔗"
	}
}

func getStatusEmoji(status string) string {
	switch strings.ToLower(status) {
	case "completed", "done", "ready":
		return "✅"
	case "in_progress", "in progress", "ongoing":
		return "🔄"
	case "blocked":
		return "🚧"
	case "pending", "waiting":
		return "⏳"
	default:
		return "📌"
	}
}
