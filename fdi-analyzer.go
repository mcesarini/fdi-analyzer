package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"os"
)

func main() {
	// Command line flags
	filePath := flag.String("file", "", "Path to the .fdi file")
	dumpSize := flag.Int("bytes", 256, "Number of bytes to dump")
	searchStr := flag.String("search", "", "Search for text (case sensitive)")
	offset := flag.Int("offset", 0, "Starting offset for reading")
	flag.Parse()

	if *filePath == "" {
		fmt.Println("Please specify a file path with -file flag")
		flag.Usage()
		return
	}

	// Read the file
	data, err := os.ReadFile(*filePath)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		return
	}

	fmt.Printf("File size: %d bytes\n", len(data))

	// Basic file analysis
	printFileHeader(data, *dumpSize, *offset)

	// Search for text if requested
	if *searchStr != "" {
		searchForText(data, *searchStr)
	}

	// Try to detect record structure
	detectRecords(data)
}

// Print the file header in hex and ASCII
func printFileHeader(data []byte, size int, offset int) {
	if offset >= len(data) {
		fmt.Println("Offset is beyond file size")
		return
	}

	end := offset + size
	if end > len(data) {
		end = len(data)
	}

	fmt.Printf("\n=== File Dump (Offset: %d) ===\n", offset)
	fmt.Println("Offset    | Hex                                             | ASCII")
	fmt.Println("----------+------------------------------------------------+------------------")

	for i := offset; i < end; i += 16 {
		rowEnd := i + 16
		if rowEnd > end {
			rowEnd = end
		}

		// Print offset
		fmt.Printf("0x%08X | ", i)

		// Print hex values
		for j := i; j < rowEnd; j++ {
			fmt.Printf("%02X ", data[j])
		}

		// Padding for incomplete rows
		for j := rowEnd; j < i+16; j++ {
			fmt.Print("   ")
		}

		fmt.Print("| ")

		// Print ASCII representation
		for j := i; j < rowEnd; j++ {
			if data[j] >= 32 && data[j] <= 126 {
				fmt.Printf("%c", data[j])
			} else {
				fmt.Print(".")
			}
		}

		fmt.Println()
	}
}

// Search for a string in the file
func searchForText(data []byte, searchStr string) {
	searchBytes := []byte(searchStr)
	fmt.Printf("\n=== Searching for: %s ===\n", searchStr)

	found := false
	for i := 0; i < len(data)-len(searchBytes)+1; i++ {
		matched := true
		for j := 0; j < len(searchBytes); j++ {
			if data[i+j] != searchBytes[j] {
				matched = false
				break
			}
		}

		if matched {
			found = true
			fmt.Printf("Found at offset: 0x%X (%d)\n", i, i)

			// Show context (16 bytes before and after)
			contextStart := i - 16
			if contextStart < 0 {
				contextStart = 0
			}

			contextEnd := i + len(searchBytes) + 16
			if contextEnd > len(data) {
				contextEnd = len(data)
			}

			fmt.Println("\nContext:")
			printFileHeader(data, contextEnd-contextStart, contextStart)
		}
	}

	if !found {
		fmt.Println("String not found in file")
	}
}

// Try to detect record structures in the file
func detectRecords(data []byte) {
	fmt.Println("\n=== Record Structure Analysis ===")

	// Look for common byte patterns that might indicate record boundaries
	repeatPatterns := make(map[string][]int)

	// Check for repeating patterns of lengths 2, 4, and 8 bytes
	for patternSize := 2; patternSize <= 8; patternSize *= 2 {
		for i := 0; i < len(data)-patternSize*2; i++ {
			pattern := data[i : i+patternSize]
			patternHex := hex.EncodeToString(pattern)

			// Look for the same pattern within the next 1000 bytes
			for j := i + patternSize; j < i+1000 && j < len(data)-patternSize+1; j++ {
				comparePattern := data[j : j+patternSize]
				if bytesEqual(pattern, comparePattern) {
					// We found a repeating pattern
					if _, exists := repeatPatterns[patternHex]; !exists {
						repeatPatterns[patternHex] = []int{i, j}
					} else {
						// Update only if this is a different occurrence
						lastPos := repeatPatterns[patternHex][len(repeatPatterns[patternHex])-1]
						if j > lastPos {
							repeatPatterns[patternHex] = append(repeatPatterns[patternHex], j)
						}
					}
					break
				}
			}
		}
	}

	// Report on potential record delimiters
	if len(repeatPatterns) > 0 {
		fmt.Println("Potential record delimiters found:")
		count := 0
		for pattern, positions := range repeatPatterns {
			if len(positions) >= 3 { // Only show patterns that repeat at least 3 times
				fmt.Printf("Pattern: 0x%s appears at offsets: ", pattern)
				for i, pos := range positions[:3] { // Show only first 3 occurrences
					if i > 0 {
						fmt.Print(", ")
					}
					fmt.Printf("0x%X", pos)
				}

				// Calculate distances between occurrences
				if len(positions) >= 2 {
					distances := make([]int, 0)
					for i := 1; i < len(positions); i++ {
						distances = append(distances, positions[i]-positions[i-1])
					}

					fmt.Print(" (Distances: ")
					for i, dist := range distances[:min(3, len(distances))] {
						if i > 0 {
							fmt.Print(", ")
						}
						fmt.Printf("%d", dist)
					}
					fmt.Print(")")
				}

				fmt.Println()
				count++

				if count >= 5 {
					fmt.Println("... and more patterns")
					break
				}
			}
		}
	} else {
		fmt.Println("No obvious repeating patterns found")
	}

	// Try to detect strings that might indicate player or team names
	fmt.Println("\nPotential text strings found:")
	stringCount := 0
	inString := false
	stringStart := 0

	for i := 0; i < len(data); i++ {
		// Look for sequences of printable ASCII or extended Latin characters
		if (data[i] >= 32 && data[i] <= 126) || (data[i] >= 192 && data[i] <= 255) {
			if !inString {
				inString = true
				stringStart = i
			}
		} else {
			if inString {
				stringLength := i - stringStart
				if stringLength >= 4 { // Only consider strings of at least 4 characters
					str := string(data[stringStart:i])
					fmt.Printf("Offset 0x%X: %s\n", stringStart, str)
					stringCount++

					if stringCount >= 10 {
						fmt.Println("... and more text strings")
						break
					}
				}
				inString = false
			}
		}
	}
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
