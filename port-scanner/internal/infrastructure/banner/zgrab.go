package banner

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"port-scanner/internal/domain"
)

// ZGrabBannerService provides banner grabbing using ZGrab2
type ZGrabBannerService struct {
	timeout time.Duration
}

// Ensure ZGrabBannerService implements BannerGrabber interface
var _ domain.BannerGrabber = (*ZGrabBannerService)(nil)

// BannerResult represents the result from ZGrab2
type BannerResult struct {
	IP     string                 `json:"ip"`
	Domain string                 `json:"domain,omitempty"`
	Data   map[string]interface{} `json:"data"`
}

// ModuleConfig defines which ZGrab2 modules to use for specific ports
type ModuleConfig struct {
	Ports    []int
	Modules  []string
	Priority int // Higher priority modules are preferred
}

// NewZGrabBannerService creates a new ZGrab2 banner service
func NewZGrabBannerService(timeout time.Duration) *ZGrabBannerService {
	return &ZGrabBannerService{
		timeout: timeout,
	}
}

// GetBanner retrieves comprehensive banner information using ZGrab2
func (z *ZGrabBannerService) GetBanner(ip string, port int) (*domain.BannerInfo, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), z.timeout)
	defer cancel()

	// Select appropriate modules based on port
	modules := z.selectModulesForPort(port)
	if len(modules) == 0 {
		// Fallback to basic banner grabbing for unknown ports
		return z.FallbackBannerGrab(ip, port)
	}

	// Build ZGrab2 command with selected modules
	cmd := z.buildZGrabCommand(ctx, ip, port, modules)

	// Execute command with proper error handling
	output, err := z.executeZGrabCommand(cmd)
	if err != nil {
		// Log the error and fall back to basic banner grabbing
		return z.FallbackBannerGrab(ip, port)
	}

	// Parse ZGrab2 output with proper result selection
	return z.parseZGrabOutput(output, port)
}

// selectModulesForPort selects appropriate ZGrab2 modules based on port number
func (z *ZGrabBannerService) selectModulesForPort(port int) []string {
	// Define port-to-module mappings with priorities
	portModules := map[int][]string{
		21:    {"ftp", "banner"},
		22:    {"ssh", "banner"},
		23:    {"telnet", "banner"},
		25:    {"smtp", "banner"},
		53:    {"banner"}, // DNS doesn't have a specific ZGrab2 module
		80:    {"http", "banner"},
		110:   {"pop3", "banner"},
		143:   {"imap", "banner"},
		443:   {"http", "tls", "banner"},
		993:   {"imap", "tls", "banner"},
		995:   {"pop3", "tls", "banner"},
		3306:  {"mysql", "banner"},
		3389:  {"banner"}, // RDP doesn't have a specific ZGrab2 module
		5432:  {"postgres", "banner"},
		6379:  {"redis", "banner"},
		27017: {"mongodb", "banner"},
		27018: {"mongodb", "banner"},
		27019: {"mongodb", "banner"},
		27020: {"mongodb", "banner"},
		1521:  {"oracle", "banner"},        // Oracle Database
		1526:  {"oracle", "banner"},        // Oracle Database (alternative)
		1433:  {"mssql", "banner"},         // Microsoft SQL Server
		1434:  {"mssql", "banner"},         // Microsoft SQL Server Browser
		5433:  {"postgres", "banner"},      // PostgreSQL (alternative)
		5434:  {"postgres", "banner"},      // PostgreSQL (alternative)
		5435:  {"postgres", "banner"},      // PostgreSQL (alternative)
		3307:  {"mysql", "banner"},         // MySQL (alternative)
		3308:  {"mysql", "banner"},         // MySQL (alternative)
		3309:  {"mysql", "banner"},         // MySQL (alternative)
		6378:  {"redis", "banner"},         // Redis (alternative)
		6380:  {"redis", "banner"},         // Redis (alternative)
		6381:  {"redis", "banner"},         // Redis (alternative)
		9200:  {"elasticsearch", "banner"}, // Elasticsearch
		9300:  {"elasticsearch", "banner"}, // Elasticsearch (cluster)
		11211: {"memcached", "banner"},     // Memcached
		11210: {"memcached", "banner"},     // Memcached (alternative)
		5984:  {"couchdb", "banner"},       // CouchDB
		5985:  {"couchdb", "banner"},       // CouchDB (alternative)
		8080:  {"http", "banner"},
		8443:  {"http", "tls", "banner"},
	}

	if modules, exists := portModules[port]; exists {
		return modules
	}

	// For unknown ports, use generic banner module
	return []string{"banner"}
}

// buildZGrabCommand builds the ZGrab2 command with selected modules
func (z *ZGrabBannerService) buildZGrabCommand(ctx context.Context, ip string, port int, modules []string) *exec.Cmd {
	args := []string{
		"--output-file", "-", // Output to stdout
		"--targets", fmt.Sprintf("%s:%d", ip, port),
		"--port", fmt.Sprintf("%d", port),
		"--timeout", fmt.Sprintf("%.0fs", z.timeout.Seconds()),
	}

	// Add selected modules
	for _, module := range modules {
		args = append(args, "--"+module)
	}

	return exec.CommandContext(ctx, "zgrab2", args...)
}

// executeZGrabCommand executes ZGrab2 command with proper error handling
func (z *ZGrabBannerService) executeZGrabCommand(cmd *exec.Cmd) ([]byte, error) {
	// Set up command with proper environment
	cmd.Stderr = nil // Suppress stderr to avoid noise

	// Execute command
	output, err := cmd.Output()
	if err != nil {
		// Check if ZGrab2 is not available
		if strings.Contains(err.Error(), "executable file not found") {
			return nil, fmt.Errorf("zgrab2 not available: %w", err)
		}

		// Check if it's a timeout error (context deadline exceeded)
		if strings.Contains(err.Error(), "context deadline exceeded") {
			return nil, fmt.Errorf("zgrab2 timeout: %w", err)
		}

		// Other execution errors
		return nil, fmt.Errorf("zgrab2 execution failed: %w", err)
	}

	return output, nil
}

// parseZGrabOutput parses the JSON output from ZGrab2 with proper result selection
func (z *ZGrabBannerService) parseZGrabOutput(output []byte, port int) (*domain.BannerInfo, error) {
	// Split output by lines (ZGrab2 outputs one JSON object per line)
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	if len(lines) == 0 {
		return &domain.BannerInfo{
			RawBanner:  "",
			Service:    "unknown",
			Protocol:   "tcp",
			Version:    "",
			Confidence: "port",
		}, nil
	}

	// Parse all results and select the best one
	var bestResult *domain.BannerInfo
	highestPriority := -1

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		var result BannerResult
		if err := json.Unmarshal([]byte(line), &result); err != nil {
			continue // Skip malformed lines
		}

		// Analyze this result
		bannerInfo := z.analyzeZGrabResult(result, port)

		// Determine priority based on confidence and module relevance
		priority := z.calculateResultPriority(bannerInfo)

		if priority > highestPriority {
			highestPriority = priority
			bestResult = bannerInfo
		}
	}

	if bestResult == nil {
		return &domain.BannerInfo{
			RawBanner:  string(output),
			Service:    "unknown",
			Protocol:   "tcp",
			Version:    "",
			Confidence: "port",
		}, nil
	}

	return bestResult, nil
}

// analyzeZGrabResult analyzes a single ZGrab2 result
func (z *ZGrabBannerService) analyzeZGrabResult(result BannerResult, port int) *domain.BannerInfo {
	// Extract banner information
	rawBanner := z.extractRawBanner(result.Data)
	service := z.identifyService(port, result.Data)
	version := z.extractVersion(result.Data)
	confidence := z.determineConfidence(result.Data)

	return &domain.BannerInfo{
		RawBanner:  rawBanner,
		Service:    service,
		Protocol:   "tcp",
		Version:    version,
		Confidence: confidence,
		Metadata:   result.Data,
	}
}

// calculateResultPriority calculates the priority of a result based on confidence and module relevance
func (z *ZGrabBannerService) calculateResultPriority(bannerInfo *domain.BannerInfo) int {
	priority := 0

	// Base priority by confidence
	switch bannerInfo.Confidence {
	case "zgrab2":
		priority += 100
	case "banner":
		priority += 50
	case "port":
		priority += 10
	}

	// Bonus for having version information
	if bannerInfo.Version != "" {
		priority += 20
	}

	// Bonus for having detailed banner
	if len(bannerInfo.RawBanner) > 10 {
		priority += 10
	}

	// Bonus for having metadata
	if len(bannerInfo.Metadata) > 0 {
		priority += 5
	}

	return priority
}

// determineConfidence determines the confidence level of the service identification
func (z *ZGrabBannerService) determineConfidence(data map[string]interface{}) string {
	// Check if we have ZGrab2 module data
	for module := range data {
		if module != "banner" && module != "ip" && module != "domain" {
			return "zgrab2"
		}
	}

	// Check if we have a meaningful banner
	if banner, ok := data["banner"].(string); ok && len(strings.TrimSpace(banner)) > 5 {
		return "banner"
	}

	return "port"
}

// extractRawBanner extracts the raw banner text from ZGrab2 data
func (z *ZGrabBannerService) extractRawBanner(data map[string]interface{}) string {
	// Try to extract banner from various ZGrab2 modules
	if banner, ok := data["banner"].(string); ok {
		return strings.TrimSpace(banner)
	}

	// Check for HTTP response
	if httpData, ok := data["http"].(map[string]interface{}); ok {
		if response, ok := httpData["response"].(map[string]interface{}); ok {
			if status, ok := response["status"].(string); ok {
				return fmt.Sprintf("HTTP %s", status)
			}
		}
	}

	// Check for SSH banner
	if sshData, ok := data["ssh"].(map[string]interface{}); ok {
		if banner, ok := sshData["server_banner"].(string); ok {
			return strings.TrimSpace(banner)
		}
	}

	// Check for TLS information
	if tlsData, ok := data["tls"].(map[string]interface{}); ok {
		if handshake, ok := tlsData["handshake_log"].(map[string]interface{}); ok {
			if serverHello, ok := handshake["server_hello"].(map[string]interface{}); ok {
				if version, ok := serverHello["version"].(string); ok {
					return fmt.Sprintf("TLS %s", version)
				}
			}
		}
	}

	// Fallback: return the entire data as JSON
	if jsonData, err := json.Marshal(data); err == nil {
		return string(jsonData)
	}

	return "No banner available"
}

// extractVersion extracts version information from ZGrab2 data
func (z *ZGrabBannerService) extractVersion(data map[string]interface{}) string {
	// Try to extract version from various ZGrab2 modules
	for module, moduleData := range data {
		if moduleMap, ok := moduleData.(map[string]interface{}); ok {
			version := z.extractVersionFromModule(module, moduleMap)
			if version != "" {
				return version
			}
		}
	}

	// Try to extract version from raw banner
	if banner, ok := data["banner"].(string); ok {
		return z.extractVersionFromBanner(banner)
	}

	return ""
}

// extractVersionFromModule extracts version from specific ZGrab2 modules
func (z *ZGrabBannerService) extractVersionFromModule(module string, data map[string]interface{}) string {
	switch module {
	case "http":
		return z.extractHTTPVersion(data)
	case "ssh":
		return z.extractSSHVersion(data)
	case "ftp":
		return z.extractFTPVersion(data)
	case "smtp":
		return z.extractSMTPVersion(data)
	case "pop3":
		return z.extractPOP3Version(data)
	case "imap":
		return z.extractIMAPVersion(data)
	case "telnet":
		return z.extractTelnetVersion(data)
	case "tls":
		return z.extractTLSVersion(data)
	case "mysql":
		return z.extractMySQLVersion(data)
	case "postgres":
		return z.extractPostgreSQLVersion(data)
	case "redis":
		return z.extractRedisVersion(data)
	case "mongodb":
		return z.extractMongoDBVersion(data)
	case "oracle":
		return z.extractOracleVersion(data)
	case "mssql":
		return z.extractMSSQLVersion(data)
	case "elasticsearch":
		return z.extractElasticsearchVersion(data)
	case "memcached":
		return z.extractMemcachedVersion(data)
	case "couchdb":
		return z.extractCouchDBVersion(data)
	case "modbus":
		return z.extractModbusVersion(data)
	case "bacnet":
		return z.extractBACnetVersion(data)
	case "fox":
		return z.extractFoxVersion(data)
	case "dnp3":
		return z.extractDNP3Version(data)
	case "ntp":
		return z.extractNTPVersion(data)
	case "s7":
		return z.extractS7Version(data)
	case "ike":
		return z.extractIKEVersion(data)
	}
	return ""
}

// extractHTTPVersion extracts version from HTTP module data
func (z *ZGrabBannerService) extractHTTPVersion(data map[string]interface{}) string {
	// Check for server header
	if response, ok := data["response"].(map[string]interface{}); ok {
		if headers, ok := response["headers"].(map[string]interface{}); ok {
			if server, ok := headers["server"].(string); ok {
				return z.extractVersionFromBanner(server)
			}
		}
	}
	return ""
}

// extractSSHVersion extracts version from SSH module data
func (z *ZGrabBannerService) extractSSHVersion(data map[string]interface{}) string {
	if banner, ok := data["server_banner"].(string); ok {
		return z.extractVersionFromBanner(banner)
	}
	return ""
}

// extractFTPVersion extracts version from FTP module data
func (z *ZGrabBannerService) extractFTPVersion(data map[string]interface{}) string {
	if banner, ok := data["banner"].(string); ok {
		return z.extractVersionFromBanner(banner)
	}
	return ""
}

// extractSMTPVersion extracts version from SMTP module data
func (z *ZGrabBannerService) extractSMTPVersion(data map[string]interface{}) string {
	if banner, ok := data["banner"].(string); ok {
		return z.extractVersionFromBanner(banner)
	}
	return ""
}

// extractPOP3Version extracts version from POP3 module data
func (z *ZGrabBannerService) extractPOP3Version(data map[string]interface{}) string {
	if banner, ok := data["banner"].(string); ok {
		return z.extractVersionFromBanner(banner)
	}
	return ""
}

// extractIMAPVersion extracts version from IMAP module data
func (z *ZGrabBannerService) extractIMAPVersion(data map[string]interface{}) string {
	if banner, ok := data["banner"].(string); ok {
		return z.extractVersionFromBanner(banner)
	}
	return ""
}

// extractTelnetVersion extracts version from Telnet module data
func (z *ZGrabBannerService) extractTelnetVersion(data map[string]interface{}) string {
	if banner, ok := data["banner"].(string); ok {
		return z.extractVersionFromBanner(banner)
	}
	return ""
}

// extractTLSVersion extracts version from TLS module data
func (z *ZGrabBannerService) extractTLSVersion(data map[string]interface{}) string {
	if handshake, ok := data["handshake_log"].(map[string]interface{}); ok {
		if serverHello, ok := handshake["server_hello"].(map[string]interface{}); ok {
			if version, ok := serverHello["version"].(string); ok {
				return version
			}
		}
	}
	return ""
}

// extractMySQLVersion extracts version from MySQL module data
func (z *ZGrabBannerService) extractMySQLVersion(data map[string]interface{}) string {
	if banner, ok := data["banner"].(string); ok {
		return z.extractVersionFromBanner(banner)
	}
	return ""
}

// extractPostgreSQLVersion extracts version from PostgreSQL module data
func (z *ZGrabBannerService) extractPostgreSQLVersion(data map[string]interface{}) string {
	if banner, ok := data["banner"].(string); ok {
		return z.extractVersionFromBanner(banner)
	}
	return ""
}

// extractRedisVersion extracts version from Redis module data
func (z *ZGrabBannerService) extractRedisVersion(data map[string]interface{}) string {
	if banner, ok := data["banner"].(string); ok {
		return z.extractVersionFromBanner(banner)
	}
	return ""
}

// extractMongoDBVersion extracts version from MongoDB module data
func (z *ZGrabBannerService) extractMongoDBVersion(data map[string]interface{}) string {
	if banner, ok := data["banner"].(string); ok {
		return z.extractVersionFromBanner(banner)
	}
	return ""
}

// extractOracleVersion extracts version from Oracle module data
func (z *ZGrabBannerService) extractOracleVersion(data map[string]interface{}) string {
	// Check for Oracle-specific fields
	if version, ok := data["version"].(string); ok {
		return version
	}
	if banner, ok := data["banner"].(string); ok {
		return z.extractVersionFromBanner(banner)
	}
	// Check for Oracle TNS protocol data
	if tns, ok := data["tns"].(map[string]interface{}); ok {
		if version, ok := tns["version"].(string); ok {
			return version
		}
	}
	return ""
}

// extractMSSQLVersion extracts version from Microsoft SQL Server module data
func (z *ZGrabBannerService) extractMSSQLVersion(data map[string]interface{}) string {
	// Check for SQL Server-specific fields
	if version, ok := data["version"].(string); ok {
		return version
	}
	if banner, ok := data["banner"].(string); ok {
		return z.extractVersionFromBanner(banner)
	}
	// Check for SQL Server Browser response
	if browser, ok := data["browser"].(map[string]interface{}); ok {
		if version, ok := browser["version"].(string); ok {
			return version
		}
	}
	return ""
}

// extractElasticsearchVersion extracts version from Elasticsearch module data
func (z *ZGrabBannerService) extractElasticsearchVersion(data map[string]interface{}) string {
	// Check for Elasticsearch-specific fields
	if version, ok := data["version"].(map[string]interface{}); ok {
		if number, ok := version["number"].(string); ok {
			return number
		}
	}
	if banner, ok := data["banner"].(string); ok {
		return z.extractVersionFromBanner(banner)
	}
	// Check for HTTP response headers
	if response, ok := data["response"].(map[string]interface{}); ok {
		if headers, ok := response["headers"].(map[string]interface{}); ok {
			if server, ok := headers["server"].(string); ok {
				return z.extractVersionFromBanner(server)
			}
		}
	}
	return ""
}

// extractMemcachedVersion extracts version from Memcached module data
func (z *ZGrabBannerService) extractMemcachedVersion(data map[string]interface{}) string {
	// Check for Memcached-specific fields
	if version, ok := data["version"].(string); ok {
		return version
	}
	if banner, ok := data["banner"].(string); ok {
		return z.extractVersionFromBanner(banner)
	}
	// Check for stats response
	if stats, ok := data["stats"].(map[string]interface{}); ok {
		if version, ok := stats["version"].(string); ok {
			return version
		}
	}
	return ""
}

// extractCouchDBVersion extracts version from CouchDB module data
func (z *ZGrabBannerService) extractCouchDBVersion(data map[string]interface{}) string {
	// Check for CouchDB-specific fields
	if version, ok := data["version"].(string); ok {
		return version
	}
	if banner, ok := data["banner"].(string); ok {
		return z.extractVersionFromBanner(banner)
	}
	// Check for HTTP response with CouchDB info
	if response, ok := data["response"].(map[string]interface{}); ok {
		if headers, ok := response["headers"].(map[string]interface{}); ok {
			if server, ok := headers["server"].(string); ok {
				return z.extractVersionFromBanner(server)
			}
		}
		// Check for JSON response body
		if body, ok := response["body"].(string); ok {
			// Try to parse JSON and extract version
			var couchInfo map[string]interface{}
			if err := json.Unmarshal([]byte(body), &couchInfo); err == nil {
				if version, ok := couchInfo["version"].(string); ok {
					return version
				}
			}
		}
	}
	return ""
}

// extractModbusVersion extracts version from Modbus module data
func (z *ZGrabBannerService) extractModbusVersion(data map[string]interface{}) string {
	if banner, ok := data["banner"].(string); ok {
		return z.extractVersionFromBanner(banner)
	}
	return ""
}

// extractBACnetVersion extracts version from BACnet module data
func (z *ZGrabBannerService) extractBACnetVersion(data map[string]interface{}) string {
	if banner, ok := data["banner"].(string); ok {
		return z.extractVersionFromBanner(banner)
	}
	return ""
}

// extractFoxVersion extracts version from Fox module data
func (z *ZGrabBannerService) extractFoxVersion(data map[string]interface{}) string {
	if banner, ok := data["banner"].(string); ok {
		return z.extractVersionFromBanner(banner)
	}
	return ""
}

// extractDNP3Version extracts version from DNP3 module data
func (z *ZGrabBannerService) extractDNP3Version(data map[string]interface{}) string {
	if banner, ok := data["banner"].(string); ok {
		return z.extractVersionFromBanner(banner)
	}
	return ""
}

// extractNTPVersion extracts version from NTP module data
func (z *ZGrabBannerService) extractNTPVersion(data map[string]interface{}) string {
	if banner, ok := data["banner"].(string); ok {
		return z.extractVersionFromBanner(banner)
	}
	return ""
}

// extractS7Version extracts version from S7 module data
func (z *ZGrabBannerService) extractS7Version(data map[string]interface{}) string {
	if banner, ok := data["banner"].(string); ok {
		return z.extractVersionFromBanner(banner)
	}
	return ""
}

// extractIKEVersion extracts version from IKE module data
func (z *ZGrabBannerService) extractIKEVersion(data map[string]interface{}) string {
	if banner, ok := data["banner"].(string); ok {
		return z.extractVersionFromBanner(banner)
	}
	return ""
}

// extractVersionFromBanner extracts version information from banner text using regex patterns
func (z *ZGrabBannerService) extractVersionFromBanner(banner string) string {
	// Database-specific version patterns (higher priority)
	dbPatterns := []string{
		// Oracle Database
		`(?i)(?:oracle|oracle\s*database)\s*([0-9]+\.[0-9]+(?:\.[0-9]+)?(?:\.[0-9]+)?)`,
		`(?i)(?:oracle)\s*([0-9]+g(?:\s*r[0-9]+)?)`,
		// Microsoft SQL Server
		`(?i)(?:microsoft\s*sql\s*server|sql\s*server)\s*([0-9]+\.[0-9]+(?:\.[0-9]+)?(?:\.[0-9]+)?)`,
		`(?i)(?:sql\s*server)\s*([0-9]{4})`,
		// Elasticsearch
		`(?i)(?:elasticsearch)\s*([0-9]+\.[0-9]+(?:\.[0-9]+)?)`,
		// Memcached
		`(?i)(?:memcached)\s*([0-9]+\.[0-9]+(?:\.[0-9]+)?)`,
		// CouchDB
		`(?i)(?:couchdb|apache\s*couchdb)\s*([0-9]+\.[0-9]+(?:\.[0-9]+)?)`,
		// Common databases
		`(?i)(?:mysql|postgresql|redis|mongodb)\s*([0-9]+\.[0-9]+(?:\.[0-9]+)?)`,
	}

	// General version patterns (lower priority)
	generalPatterns := []string{
		`(?i)(?:version|v|ver)\s*[:\s]*([0-9]+\.[0-9]+(?:\.[0-9]+)?(?:\.[0-9]+)?)`,
		`(?i)(?:openssh|ssh)\s*([0-9]+\.[0-9]+(?:\.[0-9]+)?)`,
		`(?i)(?:apache|nginx|iis)\s*[/\s]*([0-9]+\.[0-9]+(?:\.[0-9]+)?)`,
		`(?i)(?:ubuntu|debian|centos|redhat|fedora)\s*([0-9]+\.[0-9]+(?:\.[0-9]+)?)`,
		`(?i)([0-9]+\.[0-9]+(?:\.[0-9]+)?(?:\.[0-9]+)?)`,
	}

	// Try database-specific patterns first
	for _, pattern := range dbPatterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(banner)
		if len(matches) > 1 {
			return strings.TrimSpace(matches[1])
		}
	}

	// Fallback to general patterns
	for _, pattern := range generalPatterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(banner)
		if len(matches) > 1 {
			return strings.TrimSpace(matches[1])
		}
	}

	return ""
}

// identifyService identifies the service based on ZGrab2 data
func (z *ZGrabBannerService) identifyService(port int, data map[string]interface{}) string {
	// Check for specific module data
	for module := range data {
		switch module {
		case "http":
			return "http"
		case "https":
			return "https"
		case "ssh":
			return "ssh"
		case "ftp":
			return "ftp"
		case "smtp":
			return "smtp"
		case "pop3":
			return "pop3"
		case "imap":
			return "imap"
		case "telnet":
			return "telnet"
		case "tls":
			return "tls"
		case "mysql":
			return "mysql"
		case "postgres":
			return "postgresql"
		case "redis":
			return "redis"
		case "mongodb":
			return "mongodb"
		case "oracle":
			return "oracle"
		case "mssql":
			return "mssql"
		case "elasticsearch":
			return "elasticsearch"
		case "memcached":
			return "memcached"
		case "couchdb":
			return "couchdb"
		case "modbus":
			return "modbus"
		case "bacnet":
			return "bacnet"
		case "fox":
			return "fox"
		case "dnp3":
			return "dnp3"
		case "ntp":
			return "ntp"
		case "s7":
			return "s7"
		case "ike":
			return "ike"
		}
	}

	// Fallback to port-based identification
	return z.IdentifyServiceByPort(port)
}

// IdentifyServiceByPort identifies service by common port numbers
func (z *ZGrabBannerService) IdentifyServiceByPort(port int) string {
	portServices := map[int]string{
		21:    "ftp",
		22:    "ssh",
		23:    "telnet",
		25:    "smtp",
		53:    "dns",
		80:    "http",
		110:   "pop3",
		143:   "imap",
		443:   "https",
		993:   "imaps",
		995:   "pop3s",
		3306:  "mysql",
		3307:  "mysql",
		3308:  "mysql",
		3309:  "mysql",
		3389:  "rdp",
		5432:  "postgresql",
		5433:  "postgresql",
		5434:  "postgresql",
		5435:  "postgresql",
		6378:  "redis",
		6379:  "redis",
		6380:  "redis",
		6381:  "redis",
		27017: "mongodb",
		27018: "mongodb",
		27019: "mongodb",
		27020: "mongodb",
		1521:  "oracle",
		1526:  "oracle",
		1433:  "mssql",
		1434:  "mssql",
		9200:  "elasticsearch",
		9300:  "elasticsearch",
		11210: "memcached",
		11211: "memcached",
		5984:  "couchdb",
		5985:  "couchdb",
		8080:  "http-proxy",
		8443:  "https-alt",
	}

	if service, exists := portServices[port]; exists {
		return service
	}

	return "unknown"
}

// FallbackBannerGrab provides a basic banner grab when ZGrab2 is not available
func (z *ZGrabBannerService) FallbackBannerGrab(ip string, port int) (*domain.BannerInfo, error) {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", ip, port), z.timeout)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	// Set read deadline
	conn.SetReadDeadline(time.Now().Add(z.timeout))

	// Send a simple probe
	_, err = conn.Write([]byte("\r\n"))
	if err != nil {
		return nil, err
	}

	// Read response
	scanner := bufio.NewScanner(conn)
	if scanner.Scan() {
		banner := strings.TrimSpace(scanner.Text())
		version := z.extractVersionFromBanner(banner)

		return &domain.BannerInfo{
			RawBanner:  banner,
			Service:    z.IdentifyServiceByPort(port),
			Protocol:   "tcp",
			Version:    version,
			Confidence: "banner",
		}, nil
	}

	return &domain.BannerInfo{
		RawBanner:  "",
		Service:    z.IdentifyServiceByPort(port),
		Protocol:   "tcp",
		Version:    "",
		Confidence: "port",
	}, fmt.Errorf("no banner received")
}
