package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
)

func main() {
	inputFile := flag.String("input", "", "Input .vec file path")
	outputFile := flag.String("output", "", "Output .vec file path")
	maxWords := flag.Int("max", 100000, "Maximum number of words to keep")
	flag.Parse()

	if *inputFile == "" || *outputFile == "" {
		log.Fatal("Usage: reduce_vec_size -input <input.vec> -output <output.vec> -max <max_words>")
		return
	}

	fmt.Printf("Reducing %s to top %d words...\n", *inputFile, *maxWords)

	// Open input file
	inFile, err := os.Open(*inputFile)
	if err != nil {
		log.Fatalf("Failed to open input file: %v", err)
	}
	defer inFile.Close()

	// Create output file
	outFile, err := os.Create(*outputFile)
	if err != nil {
		log.Fatalf("Failed to create output file: %v", err)
	}
	defer outFile.Close()

	scanner := bufio.NewScanner(inFile)
	writer := bufio.NewWriter(outFile)
	defer func() { _ = writer.Flush() }()

	// Increase buffer size for large lines
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	// Read and parse header
	if !scanner.Scan() {
		log.Fatal("Failed to read header")
	}

	headerLine := scanner.Text()
	parts := strings.Fields(headerLine)
	if len(parts) != 2 {
		log.Fatal("Invalid header format")
	}

	dimension := parts[1]

	// Write new header with reduced word count
	newHeader := fmt.Sprintf("%d %s\n", *maxWords, dimension)
	if _, err := writer.WriteString(newHeader); err != nil {
		log.Fatalf("Failed to write header: %v", err)
	}

	// Copy first N words
	wordCount := 0
	for scanner.Scan() && wordCount < *maxWords {
		line := scanner.Text()
		if len(line) == 0 {
			continue
		}

		if _, err := writer.WriteString(line + "\n"); err != nil {
			log.Fatalf("Failed to write line: %v", err)
		}

		wordCount++
		if wordCount%10000 == 0 {
			fmt.Printf("Processed %d words...\n", wordCount)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("Error reading file: %v", err)
	}

	fmt.Printf("Successfully created reduced vector file with %d words\n", wordCount)

	// Show file sizes
	inStat, _ := os.Stat(*inputFile)
	outStat, _ := os.Stat(*outputFile)
	fmt.Printf("Original size: %.2f GB\n", float64(inStat.Size())/(1024*1024*1024))
	fmt.Printf("Reduced size: %.2f MB\n", float64(outStat.Size())/(1024*1024))
	fmt.Printf("Reduction: %.1f%%\n", (1-float64(outStat.Size())/float64(inStat.Size()))*100)
}
