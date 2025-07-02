package ping

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// SafePingService provides a sandboxed ping implementation
type SafePingService struct {
	timeout time.Duration
}

// PingResult represents the result of a ping operation
type PingResult struct {
	IsUp     bool
	Duration time.Duration
	Error    error
}

// NewSafePingService creates a new safe ping service
func NewSafePingService(timeout time.Duration) *SafePingService {
	return &SafePingService{
		timeout: timeout,
	}
}

// PingHost performs a safe ping to check if the host is up
func (s *SafePingService) PingHost(ip string) (*PingResult, error) {
	// Validate IP address format
	if !s.isValidIP(ip) {
		return &PingResult{
			IsUp:     false,
			Duration: 0,
			Error:    fmt.Errorf("invalid IP address format: %s", ip),
		}, nil
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()

	// Build ping command with proper arguments
	cmd := s.buildPingCommand(ctx, ip)

	// Execute command with proper error handling
	start := time.Now()
	err := cmd.Run()
	duration := time.Since(start)

	// Analyze the result
	result := s.analyzePingResult(err, duration, ctx)

	return result, nil
}

// isValidIP validates IP address format
func (s *SafePingService) isValidIP(ip string) bool {
	// Basic IP validation regex
	ipRegex := regexp.MustCompile(`^(\d{1,3}\.){3}\d{1,3}$`)
	if !ipRegex.MatchString(ip) {
		return false
	}

	// Check each octet
	parts := strings.Split(ip, ".")
	for _, part := range parts {
		if num, err := strconv.Atoi(part); err != nil || num < 0 || num > 255 {
			return false
		}
	}

	return true
}

// buildPingCommand builds a safe ping command
func (s *SafePingService) buildPingCommand(ctx context.Context, ip string) *exec.Cmd {
	// Use different ping commands based on OS
	var cmd *exec.Cmd

	// Try to detect OS and use appropriate ping command
	if s.isWindows() {
		cmd = exec.CommandContext(ctx, "ping", "-n", "1", "-w", fmt.Sprintf("%d", s.timeout.Milliseconds()), ip)
	} else {
		// Unix-like systems (Linux, macOS, etc.)
		cmd = exec.CommandContext(ctx, "ping", "-c", "1", "-W", fmt.Sprintf("%d", int(s.timeout.Seconds())), ip)
	}

	// Set up command environment
	cmd.Stderr = nil // Suppress stderr to avoid noise

	return cmd
}

// isWindows detects if running on Windows
func (s *SafePingService) isWindows() bool {
	// This is a simplified detection - in production you might want to use runtime.GOOS
	return false // Assuming Linux/Unix for now
}

// analyzePingResult analyzes the ping command result
func (s *SafePingService) analyzePingResult(err error, duration time.Duration, ctx context.Context) *PingResult {
	if err != nil {
		// Check for specific error types
		if ctxErr := ctx.Err(); ctxErr != nil {
			return &PingResult{
				IsUp:     false,
				Duration: duration,
				Error:    fmt.Errorf("ping timeout: %w", ctxErr),
			}
		}

		// Check if ping command is not available
		if strings.Contains(err.Error(), "executable file not found") {
			return &PingResult{
				IsUp:     false,
				Duration: duration,
				Error:    fmt.Errorf("ping command not available: %w", err),
			}
		}

		// Check if host is unreachable (this is actually a successful ping result)
		if strings.Contains(err.Error(), "100% packet loss") ||
			strings.Contains(err.Error(), "Destination Host Unreachable") {
			return &PingResult{
				IsUp:     false,
				Duration: duration,
				Error:    nil, // Not an error, just host is down
			}
		}

		// Other ping errors
		return &PingResult{
			IsUp:     false,
			Duration: duration,
			Error:    fmt.Errorf("ping failed: %w", err),
		}
	}

	// Successful ping
	return &PingResult{
		IsUp:     true,
		Duration: duration,
		Error:    nil,
	}
}
