package main

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "jbraincli",
		Short: "JosephsBrain CLI tool",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("JosephsBrain CLI v0.1.0 - Go Edition!")
			fmt.Println("Available commands: task, project, remember, context")
		},
	}

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
}

func taskCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "task",
		Short: "Task management commands",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Task management - Coming soon!")
		},
	}
	return cmd
}

func projectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project",
		Short: "Project management commands",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Project management - Coming soon!")
		},
	}
	return cmd
}

func memoryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remember",
		Short: "Memory management commands",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Memory management - Coming soon!")
		},
	}
	return cmd
}

func contextCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "context",
		Short: "Context management commands", 
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Context management - Coming soon!")
		},
	}
	return cmd
} 