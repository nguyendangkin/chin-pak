package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/nguyendangkin/progress-chin/progress"
)

func main() {
	// Manual argument parsing to handle -mb flag properly
	var maxSizeMB int
	var sources []string

	// Parse arguments manually
	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]
		if arg == "-mb" {
			if i+1 >= len(os.Args) {
				fmt.Println("Error: -mb flag requires a value")
				return
			}
			var err error
			maxSizeMB, err = strconv.Atoi(os.Args[i+1])
			if err != nil {
				fmt.Printf("Error: invalid value for -mb: %s\n", os.Args[i+1])
				return
			}
			i++ // Skip the next argument as it's the value for -mb
		} else {
			sources = append(sources, arg)
		}
	}

	if len(sources) < 1 {
		fmt.Println("Usage:")
		fmt.Println("  Compress single:     chin <file/folder>")
		fmt.Println("  Compress multiple:   chin <file1> <file2> <folder1> ...")
		fmt.Println("  Compress with split: chin -mb 1000 <file/folder>")
		fmt.Println("  Decompress:          chin <file.chin> or chin <file-1.chin>")
		return
	}

	// Check if this is decompression
	firstArg := sources[0]
	if strings.HasSuffix(firstArg, ".chin") {
		fmt.Println("Starting decompression...")
		start := time.Now()
		err := decompress(firstArg)
		duration := time.Since(start)
		if err != nil {
			fmt.Printf("\nDecompression error: %v\n", err)
		} else {
			fmt.Printf("\nDecompression completed! (%.2fs)\n", duration.Seconds())
		}
		return
	}

	// Compression mode
	fmt.Println("Starting compression...")
	start := time.Now()

	var err error
	if len(sources) == 1 {
		// Single file/folder compression
		err = compress(sources[0], maxSizeMB)
	} else {
		// Multiple files/folders compression
		err = compressMultiple(sources, maxSizeMB)
	}

	duration := time.Since(start)
	if err != nil {
		fmt.Printf("\nCompression error: %v\n", err)
	} else {
		fmt.Printf("\nCompression completed! (%.2fs)\n", duration.Seconds())
	}
}

// ------------------- Compress Single Item -------------------
func compress(src string, maxSizeMB int) error {
	src = filepath.Clean(src)
	info, err := os.Stat(src)
	if err != nil {
		return err
	}

	// Always place output in current directory
	var outFile string
	if info.IsDir() {
		outFile = filepath.Base(src) + ".chin"
	} else {
		outFile = strings.TrimSuffix(info.Name(), filepath.Ext(info.Name())) + ".chin"
	}

	if maxSizeMB > 0 {
		return compressWithSplit(src, outFile, maxSizeMB)
	}

	return compressToFile(src, outFile)
}

// ------------------- Compress Multiple Items -------------------
func compressMultiple(sources []string, maxSizeMB int) error {
	// Get the first item's name for output filename
	firstItem := filepath.Base(sources[0])
	if strings.Contains(firstItem, ".") {
		firstItem = strings.TrimSuffix(firstItem, filepath.Ext(firstItem))
	}

	// Always place output in current directory
	outFile := firstItem + "-all.chin"

	if maxSizeMB > 0 {
		return compressMultipleWithSplit(sources, outFile, maxSizeMB)
	}

	return compressMultipleToFile(sources, outFile)
}

// ------------------- Compress to Single File -------------------
func compressToFile(src, outFile string) error {
	f, err := os.Create(outFile)
	if err != nil {
		return err
	}
	defer f.Close()

	absOutFile, _ := filepath.Abs(outFile)
	baseDir := filepath.Dir(src)

	// Count total entries
	totalEntries := 0
	filepath.Walk(src, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		absPath, _ := filepath.Abs(path)
		if absPath != absOutFile {
			totalEntries++
		}
		return nil
	})
	progressBar := progress.NewProgressBar(totalEntries)

	return filepath.Walk(src, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		absPath, _ := filepath.Abs(path)
		if absPath == absOutFile {
			return nil
		}
		relPath, _ := filepath.Rel(baseDir, path)

		if err := writeEntry(f, relPath, path, info); err != nil {
			return err
		}

		progressBar.Increment()
		return nil
	})
}

// ------------------- Compress Multiple Items to Single File -------------------
func compressMultipleToFile(sources []string, outFile string) error {
	f, err := os.Create(outFile)
	if err != nil {
		return err
	}
	defer f.Close()

	absOutFile, _ := filepath.Abs(outFile)

	// Count total entries from all sources
	totalEntries := 0
	for _, src := range sources {
		filepath.Walk(src, func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return err
			}
			absPath, _ := filepath.Abs(path)
			if absPath != absOutFile {
				totalEntries++
			}
			return nil
		})
	}

	progressBar := progress.NewProgressBar(totalEntries)

	// Process each source - keep original structure
	for _, src := range sources {
		// For multiple sources, use the source path as-is to preserve structure
		srcInfo, err := os.Stat(src)
		if err != nil {
			continue
		}

		if srcInfo.IsDir() {
			// For directories, walk and preserve the full path structure
			err := filepath.Walk(src, func(path string, info fs.FileInfo, err error) error {
				if err != nil {
					return err
				}
				absPath, _ := filepath.Abs(path)
				if absPath == absOutFile {
					return nil
				}

				// Use the full path relative to current directory to preserve structure
				relPath, _ := filepath.Rel(".", path)

				if err := writeEntry(f, relPath, path, info); err != nil {
					return err
				}

				progressBar.Increment()
				return nil
			})
			if err != nil {
				return err
			}
		} else {
			// For files, use the path relative to current directory
			relPath, _ := filepath.Rel(".", src)
			if err := writeEntry(f, relPath, src, srcInfo); err != nil {
				return err
			}
			progressBar.Increment()
		}
	}

	return nil
}

// ------------------- Compress with Split -------------------
func compressWithSplit(src, outFile string, maxSizeMB int) error {
	// Create temporary buffer to collect all data
	var buf bytes.Buffer
	absOutFile, _ := filepath.Abs(outFile)
	baseDir := filepath.Dir(src)

	// Count total entries
	totalEntries := 0
	filepath.Walk(src, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		absPath, _ := filepath.Abs(path)
		if absPath != absOutFile {
			totalEntries++
		}
		return nil
	})
	progressBar := progress.NewProgressBar(totalEntries)

	// Collect all data first
	err := filepath.Walk(src, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		absPath, _ := filepath.Abs(path)
		if absPath == absOutFile {
			return nil
		}
		relPath, _ := filepath.Rel(baseDir, path)

		if err := writeEntry(&buf, relPath, path, info); err != nil {
			return err
		}

		progressBar.Increment()
		return nil
	})

	if err != nil {
		return err
	}

	// Split the data into multiple files
	return splitDataToFiles(buf.Bytes(), outFile, maxSizeMB)
}

// ------------------- Compress Multiple with Split -------------------
func compressMultipleWithSplit(sources []string, outFile string, maxSizeMB int) error {
	// Create temporary buffer to collect all data
	var buf bytes.Buffer
	absOutFile, _ := filepath.Abs(outFile)

	// Count total entries from all sources
	totalEntries := 0
	for _, src := range sources {
		filepath.Walk(src, func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return err
			}
			absPath, _ := filepath.Abs(path)
			if absPath != absOutFile {
				totalEntries++
			}
			return nil
		})
	}

	progressBar := progress.NewProgressBar(totalEntries)

	// Process each source and collect data - preserve structure
	for _, src := range sources {
		srcInfo, err := os.Stat(src)
		if err != nil {
			continue
		}

		if srcInfo.IsDir() {
			// For directories, walk and preserve the full path structure
			err := filepath.Walk(src, func(path string, info fs.FileInfo, err error) error {
				if err != nil {
					return err
				}
				absPath, _ := filepath.Abs(path)
				if absPath == absOutFile {
					return nil
				}

				// Use the full path relative to current directory to preserve structure
				relPath, _ := filepath.Rel(".", path)

				if err := writeEntry(&buf, relPath, path, info); err != nil {
					return err
				}

				progressBar.Increment()
				return nil
			})
			if err != nil {
				return err
			}
		} else {
			// For files, use the path relative to current directory
			relPath, _ := filepath.Rel(".", src)
			if err := writeEntry(&buf, relPath, src, srcInfo); err != nil {
				return err
			}
			progressBar.Increment()
		}
	}

	// Split the data into multiple files
	return splitDataToFiles(buf.Bytes(), outFile, maxSizeMB)
}

// ------------------- Helper Functions -------------------
func writeEntry(writer interface{}, relPath, fullPath string, info fs.FileInfo) error {
	relPathBytes := []byte(relPath)
	pathLen := uint16(len(relPathBytes))

	if f, ok := writer.(*os.File); ok {
		binary.Write(f, binary.LittleEndian, pathLen)
		f.Write(relPathBytes)

		if info.IsDir() {
			var dataLen uint32 = 0
			binary.Write(f, binary.LittleEndian, dataLen)
		} else {
			data, err := os.ReadFile(fullPath)
			if err != nil {
				return err
			}
			dataLen := uint32(len(data))
			binary.Write(f, binary.LittleEndian, dataLen)
			f.Write(data)
		}
	} else if buf, ok := writer.(*bytes.Buffer); ok {
		binary.Write(buf, binary.LittleEndian, pathLen)
		buf.Write(relPathBytes)

		if info.IsDir() {
			var dataLen uint32 = 0
			binary.Write(buf, binary.LittleEndian, dataLen)
		} else {
			data, err := os.ReadFile(fullPath)
			if err != nil {
				return err
			}
			dataLen := uint32(len(data))
			binary.Write(buf, binary.LittleEndian, dataLen)
			buf.Write(data)
		}
	}

	return nil
}

func splitDataToFiles(data []byte, outFile string, maxSizeMB int) error {
	maxSize := maxSizeMB * 1024 * 1024 // Convert MB to bytes
	totalSize := len(data)
	parts := (totalSize + maxSize - 1) / maxSize // Ceiling division

	baseFileName := strings.TrimSuffix(outFile, ".chin")

	for i := 0; i < parts; i++ {
		partFileName := fmt.Sprintf("%s-%d.chin", baseFileName, i+1)

		start := i * maxSize
		end := start + maxSize
		if end > totalSize {
			end = totalSize
		}

		partData := data[start:end]
		if err := os.WriteFile(partFileName, partData, 0644); err != nil {
			return err
		}

		fmt.Printf("\nCreated part %d/%d: %s (%.2f MB)\n", i+1, parts, partFileName, float64(len(partData))/1024/1024)
	}

	return nil
}

// ------------------- Decompress Data -------------------
func decompress(src string) error {
	// Check if this is a split file (ends with -1.chin, -2.chin, etc.)
	if matched, _ := regexp.MatchString(`-\d+\.chin$`, src); matched {
		return decompressSplit(src)
	}

	return decompressRegular(src)
}

func decompressRegular(src string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	return processDecompression(data)
}

func decompressSplit(src string) error {
	// Extract base filename and find all parts
	re := regexp.MustCompile(`(.+)-(\d+)\.chin$`)
	matches := re.FindStringSubmatch(src)
	if len(matches) != 3 {
		return fmt.Errorf("invalid split file format")
	}

	baseFileName := matches[1]
	dir := filepath.Dir(src)

	// Find all part files
	var partFiles []string
	var partNumbers []int

	files, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	partPattern := regexp.MustCompile(`^` + regexp.QuoteMeta(filepath.Base(baseFileName)) + `-(\d+)\.chin$`)
	for _, file := range files {
		if matches := partPattern.FindStringSubmatch(file.Name()); matches != nil {
			partNumber, err := strconv.Atoi(matches[1])
			if err != nil {
				continue
			}
			partFiles = append(partFiles, filepath.Join(dir, file.Name()))
			partNumbers = append(partNumbers, partNumber)
		}
	}

	if len(partFiles) == 0 {
		return fmt.Errorf("no part files found for %s", baseFileName)
	}

	// Sort part files by number
	sort.Slice(partFiles, func(i, j int) bool {
		re := regexp.MustCompile(`-(\d+)\.chin$`)
		numI, _ := strconv.Atoi(re.FindStringSubmatch(partFiles[i])[1])
		numJ, _ := strconv.Atoi(re.FindStringSubmatch(partFiles[j])[1])
		return numI < numJ
	})

	// Sort part numbers to check for missing parts
	sort.Ints(partNumbers)

	// Check for missing parts
	fmt.Printf("Found %d part files\n", len(partFiles))

	// Verify all parts are consecutive starting from 1
	for i, partNum := range partNumbers {
		expectedNum := i + 1
		if partNum != expectedNum {
			if partNum > expectedNum {
				// Missing parts before this one
				var missingParts []int
				for j := expectedNum; j < partNum; j++ {
					missingParts = append(missingParts, j)
				}
				return fmt.Errorf("missing part files: %s-%v.chin", filepath.Base(baseFileName), missingParts)
			}
		}
	}

	// Check if we might be missing parts at the end by examining the last part
	// This is a heuristic check - if the last part is suspiciously large, we might be missing parts
	lastPartFile := partFiles[len(partFiles)-1]
	lastPartInfo, err := os.Stat(lastPartFile)
	if err != nil {
		return fmt.Errorf("cannot stat last part file: %v", err)
	}

	// If we have multiple parts, check if all parts (except possibly the last) are similar in size
	if len(partFiles) > 1 {
		firstPartInfo, err := os.Stat(partFiles[0])
		if err != nil {
			return fmt.Errorf("cannot stat first part file: %v", err)
		}

		firstPartSize := firstPartInfo.Size()
		lastPartSize := lastPartInfo.Size()

		// If the last part is significantly larger than the first part,
		// it might indicate missing parts (this is just a warning, not an error)
		if lastPartSize > firstPartSize*2 {
			fmt.Printf("Warning: Last part (%s) is significantly larger than first part.\n", filepath.Base(lastPartFile))
			fmt.Printf("This might indicate missing part files. Proceeding with available parts...\n")
		}
	}

	// Additional check: Try to read a small portion of each part to ensure they're valid
	for i, partFile := range partFiles {
		file, err := os.Open(partFile)
		if err != nil {
			return fmt.Errorf("cannot open part file %s: %v", filepath.Base(partFile), err)
		}

		// Try to read first few bytes to ensure file is readable
		buffer := make([]byte, 10)
		_, err = file.Read(buffer)
		file.Close()

		if err != nil && err.Error() != "EOF" {
			return fmt.Errorf("cannot read part file %s: %v", filepath.Base(partFile), err)
		}

		fmt.Printf("Verified part %d/%d: %s\n", i+1, len(partFiles), filepath.Base(partFile))
	}

	// Combine all parts
	var combinedData []byte
	for i, partFile := range partFiles {
		fmt.Printf("Reading part %d/%d: %s\n", i+1, len(partFiles), filepath.Base(partFile))
		data, err := os.ReadFile(partFile)
		if err != nil {
			return fmt.Errorf("failed to read part file %s: %v", filepath.Base(partFile), err)
		}
		combinedData = append(combinedData, data...)
	}

	fmt.Printf("Combined data size: %.2f MB\n", float64(len(combinedData))/1024/1024)

	return processDecompression(combinedData)
}

func processDecompression(data []byte) error {
	buf := bytes.NewReader(data)
	totalEntries := 0

	// Count total entries with error handling
	for {
		var pathLen uint16
		if err := binary.Read(buf, binary.LittleEndian, &pathLen); err != nil {
			break
		}

		// Check if we can read the path
		if int64(pathLen) > int64(buf.Len()) {
			return fmt.Errorf("corrupted data: path length %d exceeds remaining data %d", pathLen, buf.Len())
		}

		buf.Seek(int64(pathLen), 1)

		var dataLen uint32
		if err := binary.Read(buf, binary.LittleEndian, &dataLen); err != nil {
			break
		}

		// Check if we can read the file data
		if int64(dataLen) > int64(buf.Len()) {
			return fmt.Errorf("corrupted data: file data length %d exceeds remaining data %d", dataLen, buf.Len())
		}

		buf.Seek(int64(dataLen), 1)
		totalEntries++
	}

	if totalEntries == 0 {
		return fmt.Errorf("no valid entries found in the archive data")
	}

	buf = bytes.NewReader(data)
	progressBar := progress.NewProgressBar(totalEntries)

	entriesProcessed := 0
	for {
		var pathLen uint16
		if err := binary.Read(buf, binary.LittleEndian, &pathLen); err != nil {
			break
		}

		pathBytes := make([]byte, pathLen)
		n, err := buf.Read(pathBytes)
		if err != nil || n != int(pathLen) {
			return fmt.Errorf("failed to read path data: expected %d bytes, got %d", pathLen, n)
		}
		relPath := string(pathBytes)

		var dataLen uint32
		if err := binary.Read(buf, binary.LittleEndian, &dataLen); err != nil {
			return fmt.Errorf("failed to read data length for path %s", relPath)
		}

		if dataLen == 0 {
			// Directory
			if err := os.MkdirAll(relPath, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %v", relPath, err)
			}
		} else {
			// File
			fileData := make([]byte, dataLen)
			n, err := buf.Read(fileData)
			if err != nil || n != int(dataLen) {
				return fmt.Errorf("failed to read file data for %s: expected %d bytes, got %d", relPath, dataLen, n)
			}

			if err := os.MkdirAll(filepath.Dir(relPath), 0755); err != nil {
				return fmt.Errorf("failed to create parent directory for %s: %v", relPath, err)
			}

			if err := os.WriteFile(relPath, fileData, 0644); err != nil {
				return fmt.Errorf("failed to write file %s: %v", relPath, err)
			}
		}

		progressBar.Increment()
		entriesProcessed++
	}

	progressBar.Finish()

	if entriesProcessed != totalEntries {
		fmt.Printf("Warning: Expected %d entries but processed %d entries\n", totalEntries, entriesProcessed)
	}

	return nil
}
