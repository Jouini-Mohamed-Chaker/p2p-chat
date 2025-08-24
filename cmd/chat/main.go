package main

import (
	"bufio"
	"log"
	"os"
	"strings"

	"github.com/Jouini-Mohamed-Chaker/p2p-chat/pkg/ui"
)

// loadEnv loads environment variables from a .env file
func loadEnv(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		// Split on first '=' only
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		
		// Remove quotes if present
		if len(value) >= 2 && ((value[0] == '"' && value[len(value)-1] == '"') ||
			(value[0] == '\'' && value[len(value)-1] == '\'')) {
			value = value[1 : len(value)-1]
		}
		
		// Set environment variable
		os.Setenv(key, value)
	}
	
	return scanner.Err()
}

func main() {
	// Load environment variables from .env file
	if err := loadEnv(".env"); err != nil {
		log.Printf("Warning: Could not load .env file: %v", err)
		log.Println("Using system environment variables...")
	} else {
		log.Println("Successfully loaded .env file")
	}

	// Optional: Log which credentials are available (without showing actual values)
	if os.Getenv("OPENRELAY_API_KEY") != "" {
		log.Println("✓ OpenRelay API key found - will use dynamic credentials")
	} else if os.Getenv("OPENRELAY_USERNAME") != "" && os.Getenv("OPENRELAY_CREDENTIAL") != "" {
		log.Println("✓ OpenRelay static credentials found")
	} else {
		log.Println("⚠ No OpenRelay credentials found - will use STUN only")
	}

	// Start your chat app
	app := ui.NewChatApp()
	app.Run()
}