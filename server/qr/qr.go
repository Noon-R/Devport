package qr

import (
	"fmt"
	"strings"

	"github.com/skip2/go-qrcode"
)

// PrintTerminalQR prints a QR code to the terminal
func PrintTerminalQR(url string) error {
	qr, err := qrcode.New(url, qrcode.Medium)
	if err != nil {
		return err
	}

	// Convert to bitmap
	bitmap := qr.Bitmap()
	size := len(bitmap)

	// Build string representation using Unicode block characters
	var sb strings.Builder

	// Top border
	sb.WriteString("\n")

	// Process two rows at a time for better aspect ratio
	for y := 0; y < size; y += 2 {
		sb.WriteString("  ") // Left margin
		for x := 0; x < size; x++ {
			top := bitmap[y][x]
			bottom := false
			if y+1 < size {
				bottom = bitmap[y+1][x]
			}

			// Use Unicode block characters
			// ▀ (upper half block), ▄ (lower half block), █ (full block), " " (empty)
			if top && bottom {
				sb.WriteString("█")
			} else if top && !bottom {
				sb.WriteString("▀")
			} else if !top && bottom {
				sb.WriteString("▄")
			} else {
				sb.WriteString(" ")
			}
		}
		sb.WriteString("\n")
	}

	fmt.Print(sb.String())
	return nil
}

// PrintStartupBanner prints the startup banner with connection URLs
func PrintStartupBanner(localURL, remoteURL string) {
	fmt.Println()
	fmt.Println("╭─────────────────────────────────────────────────────╮")
	fmt.Println("│                                                     │")
	fmt.Println("│   🚀 Devport is running                             │")
	fmt.Println("│                                                     │")
	fmt.Printf("│   Local:  %-40s│\n", localURL)
	if remoteURL != "" {
		fmt.Printf("│   Remote: %-40s│\n", remoteURL)
		fmt.Println("│                                                     │")
		fmt.Println("│   Scan the QR code below to connect:               │")
		fmt.Println("│                                                     │")
	}
	fmt.Println("╰─────────────────────────────────────────────────────╯")

	if remoteURL != "" {
		if err := PrintTerminalQR(remoteURL); err != nil {
			fmt.Printf("  (Failed to generate QR code: %v)\n", err)
		}
		fmt.Println()
	}
}
