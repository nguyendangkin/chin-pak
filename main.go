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

// ANSI color codes for beautiful terminal output
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorPurple = "\033[35m"
	ColorCyan   = "\033[36m"
	ColorWhite  = "\033[37m"
	ColorBold   = "\033[1m"
	ColorDim    = "\033[2m"
)

// Beautiful logging functions
func logInfo(message string, args ...interface{}) {
	fmt.Printf("%s[INFO]%s %s%s%s\n", ColorBlue+ColorBold, ColorReset, ColorWhite, fmt.Sprintf(message, args...), ColorReset)
}

func logSuccess(message string, args ...interface{}) {
	fmt.Printf("%s[SUCCESS]%s %s%s%s\n", ColorGreen+ColorBold, ColorReset, ColorGreen, fmt.Sprintf(message, args...), ColorReset)
}

func logWarning(message string, args ...interface{}) {
	fmt.Printf("%s[WARNING]%s %s%s%s\n", ColorYellow+ColorBold, ColorReset, ColorYellow, fmt.Sprintf(message, args...), ColorReset)
}

func logError(message string, args ...interface{}) {
	fmt.Printf("%s[ERROR]%s %s%s%s\n", ColorRed+ColorBold, ColorReset, ColorRed, fmt.Sprintf(message, args...), ColorReset)
}

func logHeader(message string, args ...interface{}) {
	fmt.Printf("\n%s%s═══════════════════════════════════════════════════════════════%s\n", ColorCyan+ColorBold, ColorWhite, ColorReset)
	fmt.Printf("%s%s  %s%s\n", ColorCyan+ColorBold, ColorWhite, fmt.Sprintf(message, args...), ColorReset)
	fmt.Printf("%s%s═══════════════════════════════════════════════════════════════%s\n\n", ColorCyan+ColorBold, ColorWhite, ColorReset)
}

func logSubHeader(message string, args ...interface{}) {
	fmt.Printf("\n%s%s── %s ──%s\n", ColorPurple+ColorBold, ColorWhite, fmt.Sprintf(message, args...), ColorReset)
}

func logDetail(label, value string) {
	fmt.Printf("%s  ▸ %s:%s %s%s%s\n", ColorDim, label, ColorReset, ColorWhite, value, ColorReset)
}

func printUsage() {
	logHeader("🗜️  CHIN COMPRESSOR - Usage Guide")

	fmt.Printf("%s  📦 Compress single:%s     chin <file/folder>\n", ColorGreen+ColorBold, ColorReset)
	fmt.Printf("%s  📦 Compress multiple:%s   chin <file1> <file2> <folder1> ...\n", ColorGreen+ColorBold, ColorReset)
	fmt.Printf("%s  📦 Compress with split:%s chin -mb 1000 <file/folder>\n", ColorGreen+ColorBold, ColorReset)
	fmt.Printf("%s  📂 Decompress:%s          chin <file.chin> or chin <file-1.chin>\n", ColorBlue+ColorBold, ColorReset)

	fmt.Printf("\n%s  Options:%s\n", ColorYellow+ColorBold, ColorReset)
	fmt.Printf("    %s-mb <size>%s  Split output into chunks of <size> MB\n", ColorCyan, ColorReset)
	fmt.Println()
}

func formatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func formatDuration(d time.Duration) string {
	if d.Seconds() < 1 {
		return fmt.Sprintf("%.0fms", d.Seconds()*1000)
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}

func main() {
	// Manual argument parsing to handle -mb flag properly
	var maxSizeMB int
	var sources []string

	// Parse arguments manually
	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]
		if arg == "-mb" {
			if i+1 >= len(os.Args) {
				logError("Flag -mb requires a value")
				return
			}
			var err error
			maxSizeMB, err = strconv.Atoi(os.Args[i+1])
			if err != nil {
				logError("Invalid value for -mb: %s", os.Args[i+1])
				return
			}
			i++ // Skip the next argument as it's the value for -mb
		} else {
			sources = append(sources, arg)
		}
	}

	if len(sources) < 1 {
		printUsage()
		return
	}

	// Check if this is decompression
	firstArg := sources[0]
	if strings.HasSuffix(firstArg, ".chin") {
		logHeader("📂 DECOMPRESSION MODE")
		logDetail("Input file", firstArg)

		start := time.Now()
		err := decompress(firstArg)
		duration := time.Since(start)

		if err != nil {
			logError("Decompression failed: %v", err)
		} else {
			logSuccess("✅ Decompression completed successfully!")
			logDetail("Duration", formatDuration(duration))
		}
		return
	}

	// Compression mode
	logHeader("📦 COMPRESSION MODE")

	if maxSizeMB > 0 {
		logDetail("Split size", fmt.Sprintf("%d MB", maxSizeMB))
	}

	if len(sources) == 1 {
		logDetail("Source", sources[0])
	} else {
		logDetail("Sources", fmt.Sprintf("%d items", len(sources)))
		for i, src := range sources {
			fmt.Printf("%s    %d. %s%s\n", ColorDim, i+1, src, ColorReset)
		}
	}

	start := time.Now()
	var err error

	if len(sources) == 1 {
		err = compress(sources[0], maxSizeMB)
	} else {
		err = compressMultiple(sources, maxSizeMB)
	}

	duration := time.Since(start)

	if err != nil {
		logError("Compression failed: %v", err)
	} else {
		logSuccess("✅ Compression completed successfully!")
		logDetail("Duration", formatDuration(duration))
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

	logSubHeader("📁 Analyzing source")
	if info.IsDir() {
		logDetail("Type", "Directory")
	} else {
		logDetail("Type", "File")
		logDetail("Size", formatFileSize(info.Size()))
	}
	logDetail("Output", outFile)

	if maxSizeMB > 0 {
		logInfo("🔄 Starting compression with splitting...")
		return compressWithSplit(src, outFile, maxSizeMB)
	}

	logInfo("🔄 Starting compression...")
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

	logSubHeader("📁 Analyzing sources")
	logDetail("Output", outFile)

	if maxSizeMB > 0 {
		logInfo("🔄 Starting multi-source compression with splitting...")
		return compressMultipleWithSplit(sources, outFile, maxSizeMB)
	}

	logInfo("🔄 Starting multi-source compression...")
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

	logInfo("📊 Found %d entries to compress", totalEntries)
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

	logInfo("📊 Found %d entries across all sources", totalEntries)
	progressBar := progress.NewProgressBar(totalEntries)

	// Process each source - keep original structure
	for i, src := range sources {
		// Only log once at the beginning, not for each source
		if i == 0 {
			logInfo("🔄 Processing all sources...")
		}

		srcInfo, err := os.Stat(src)
		if err != nil {
			logWarning("Cannot access source: %s", src)
			continue
		}

		if srcInfo.IsDir() {
			err := filepath.Walk(src, func(path string, info fs.FileInfo, err error) error {
				if err != nil {
					return err
				}
				absPath, _ := filepath.Abs(path)
				if absPath == absOutFile {
					return nil
				}

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
	logInfo("📦 Collecting data for splitting...")

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

	logInfo("📊 Found %d entries to compress", totalEntries)
	progressBar := progress.NewProgressBar(totalEntries)

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

	logInfo("✂️ Splitting data into %d MB chunks...", maxSizeMB)
	return splitDataToFiles(buf.Bytes(), outFile, maxSizeMB)
}

// ------------------- Compress Multiple with Split -------------------
func compressMultipleWithSplit(sources []string, outFile string, maxSizeMB int) error {
	logInfo("📦 Collecting data from multiple sources...")

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

	logInfo("📊 Found %d entries across all sources", totalEntries)
	progressBar := progress.NewProgressBar(totalEntries)

	for i, src := range sources {
		// Only log once at the beginning, not for each source
		if i == 0 {
			logInfo("🔄 Processing all sources...")
		}

		srcInfo, err := os.Stat(src)
		if err != nil {
			logWarning("Cannot access source: %s", src)
			continue
		}

		if srcInfo.IsDir() {
			err := filepath.Walk(src, func(path string, info fs.FileInfo, err error) error {
				if err != nil {
					return err
				}
				absPath, _ := filepath.Abs(path)
				if absPath == absOutFile {
					return nil
				}

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
			relPath, _ := filepath.Rel(".", src)
			if err := writeEntry(&buf, relPath, src, srcInfo); err != nil {
				return err
			}
			progressBar.Increment()
		}
	}

	logInfo("✂️ Splitting data into %d MB chunks...", maxSizeMB)
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
	maxSize := maxSizeMB * 1024 * 1024
	totalSize := len(data)
	parts := (totalSize + maxSize - 1) / maxSize

	baseFileName := strings.TrimSuffix(outFile, ".chin")

	logSubHeader("✂️ Creating split files")
	logDetail("Total size", formatFileSize(int64(totalSize)))
	logDetail("Parts to create", fmt.Sprintf("%d", parts))

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

		logSuccess("📦 Created part %d/%d: %s (%s)",
			i+1, parts, partFileName, formatFileSize(int64(len(partData))))
	}

	return nil
}

// ------------------- Decompress Data -------------------
func decompress(src string) error {
	// Check if this is a split file
	if matched, _ := regexp.MatchString(`-\d+\.chin$`, src); matched {
		logInfo("🔍 Detected split archive format")
		return decompressSplit(src)
	}

	logInfo("🔍 Detected regular archive format")
	return decompressRegular(src)
}

func decompressRegular(src string) error {
	logInfo("📖 Reading archive file...")
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	logDetail("Archive size", formatFileSize(int64(len(data))))
	return processDecompression(data)
}

func decompressSplit(src string) error {
	logSubHeader("🔍 Locating split archive parts")

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

	sort.Ints(partNumbers)

	logSuccess("🔍 Found %d part files", len(partFiles))

	// Check for missing parts
	for i, partNum := range partNumbers {
		expectedNum := i + 1
		if partNum != expectedNum {
			if partNum > expectedNum {
				var missingParts []int
				for j := expectedNum; j < partNum; j++ {
					missingParts = append(missingParts, j)
				}
				return fmt.Errorf("missing part files: %s-%v.chin", filepath.Base(baseFileName), missingParts)
			}
		}
	}

	// Verify parts integrity
	logInfo("🔍 Verifying parts integrity...")
	for i, partFile := range partFiles {
		file, err := os.Open(partFile)
		if err != nil {
			return fmt.Errorf("cannot open part file %s: %v", filepath.Base(partFile), err)
		}

		buffer := make([]byte, 10)
		_, err = file.Read(buffer)
		file.Close()

		if err != nil && err.Error() != "EOF" {
			return fmt.Errorf("cannot read part file %s: %v", filepath.Base(partFile), err)
		}

		info, _ := os.Stat(partFile)
		logDetail(fmt.Sprintf("Part %d", i+1), fmt.Sprintf("%s (%s)",
			filepath.Base(partFile), formatFileSize(info.Size())))
	}

	// Combine all parts with single progress bar
	logInfo("🔗 Combining archive parts...")
	var combinedData []byte
	var totalSize int64

	// Create progress bar for combining parts
	progressBar := progress.NewProgressBar(len(partFiles))

	for _, partFile := range partFiles {
		data, err := os.ReadFile(partFile)
		if err != nil {
			return fmt.Errorf("failed to read part file %s: %v", filepath.Base(partFile), err)
		}
		combinedData = append(combinedData, data...)
		totalSize += int64(len(data))
		progressBar.Increment()
	}

	logSuccess("🔗 Combined archive size: %s", formatFileSize(totalSize))

	return processDecompression(combinedData)
}

func processDecompression(data []byte) error {
	logInfo("📊 Analyzing archive structure...")

	buf := bytes.NewReader(data)
	totalEntries := 0

	// Count total entries with error handling
	for {
		var pathLen uint16
		if err := binary.Read(buf, binary.LittleEndian, &pathLen); err != nil {
			break
		}

		if int64(pathLen) > int64(buf.Len()) {
			return fmt.Errorf("corrupted data: path length %d exceeds remaining data %d", pathLen, buf.Len())
		}

		buf.Seek(int64(pathLen), 1)

		var dataLen uint32
		if err := binary.Read(buf, binary.LittleEndian, &dataLen); err != nil {
			break
		}

		if int64(dataLen) > int64(buf.Len()) {
			return fmt.Errorf("corrupted data: file data length %d exceeds remaining data %d", dataLen, buf.Len())
		}

		buf.Seek(int64(dataLen), 1)
		totalEntries++
	}

	if totalEntries == 0 {
		return fmt.Errorf("no valid entries found in the archive data")
	}

	logSuccess("📊 Found %d entries to extract", totalEntries)

	buf = bytes.NewReader(data)
	progressBar := progress.NewProgressBar(totalEntries)

	entriesProcessed := 0
	var extractedFiles, extractedDirs int

	logInfo("📂 Extracting archive contents...")

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
			extractedDirs++
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
			extractedFiles++
		}

		progressBar.Increment()
		entriesProcessed++
	}

	progressBar.Finish()

	if entriesProcessed != totalEntries {
		logWarning("Expected %d entries but processed %d entries", totalEntries, entriesProcessed)
	}

	logSubHeader("📈 Extraction Summary")
	logDetail("Files extracted", fmt.Sprintf("%d", extractedFiles))
	logDetail("Directories created", fmt.Sprintf("%d", extractedDirs))
	logDetail("Total entries", fmt.Sprintf("%d", entriesProcessed))

	return nil
}
