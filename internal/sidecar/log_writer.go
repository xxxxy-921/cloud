package sidecar

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

const (
	maxLogFileSize = 10 * 1024 * 1024 // 10MB
	maxLogBackups  = 3
	ringBufferSize = 4096 // number of log lines in ring buffer
)

// LogWriter implements io.Writer and captures process output to a local
// rotating log file and an in-memory ring buffer for batch upload.
type LogWriter struct {
	processName string
	stream      string // "stdout" or "stderr"
	logDir      string

	mu       sync.Mutex
	file     *os.File
	fileSize int64
	ring     []LogLine
	ringPos  int
	ringFull bool
}

// LogLine is a single captured log line.
type LogLine struct {
	ProcessName string `json:"processName"`
	Stream      string `json:"stream"`
	Content     string `json:"content"`
}

func NewLogWriter(logDir, processName, stream string) (*LogWriter, error) {
	dir := filepath.Join(logDir, processName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create log dir: %w", err)
	}

	filePath := filepath.Join(dir, stream+".log")
	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("open log file: %w", err)
	}

	info, _ := f.Stat()
	size := int64(0)
	if info != nil {
		size = info.Size()
	}

	return &LogWriter{
		processName: processName,
		stream:      stream,
		logDir:      dir,
		file:        f,
		fileSize:    size,
		ring:        make([]LogLine, ringBufferSize),
	}, nil
}

func (lw *LogWriter) Write(p []byte) (int, error) {
	lw.mu.Lock()
	defer lw.mu.Unlock()

	// Write to file
	n, err := lw.file.Write(p)
	if err != nil {
		return n, err
	}
	lw.fileSize += int64(n)

	// Add to ring buffer
	line := LogLine{
		ProcessName: lw.processName,
		Stream:      lw.stream,
		Content:     string(p),
	}
	lw.ring[lw.ringPos] = line
	lw.ringPos = (lw.ringPos + 1) % ringBufferSize
	if lw.ringPos == 0 {
		lw.ringFull = true
	}

	// Rotate if needed
	if lw.fileSize >= maxLogFileSize {
		lw.rotate()
	}

	return n, nil
}

// Drain returns all buffered log lines and clears the buffer.
func (lw *LogWriter) Drain() []LogLine {
	lw.mu.Lock()
	defer lw.mu.Unlock()

	var lines []LogLine
	if lw.ringFull {
		// Read from ringPos to end, then from 0 to ringPos
		lines = make([]LogLine, ringBufferSize)
		copy(lines, lw.ring[lw.ringPos:])
		copy(lines[ringBufferSize-lw.ringPos:], lw.ring[:lw.ringPos])
	} else if lw.ringPos > 0 {
		lines = make([]LogLine, lw.ringPos)
		copy(lines, lw.ring[:lw.ringPos])
	}

	// Reset buffer
	lw.ringPos = 0
	lw.ringFull = false

	return lines
}

func (lw *LogWriter) Close() error {
	lw.mu.Lock()
	defer lw.mu.Unlock()
	if lw.file != nil {
		return lw.file.Close()
	}
	return nil
}

func (lw *LogWriter) rotate() {
	lw.file.Close()

	base := filepath.Join(lw.logDir, lw.stream+".log")

	// Remove oldest backup
	oldest := fmt.Sprintf("%s.%d", base, maxLogBackups)
	os.Remove(oldest)

	// Shift backups
	for i := maxLogBackups - 1; i >= 1; i-- {
		src := fmt.Sprintf("%s.%d", base, i)
		dst := fmt.Sprintf("%s.%d", base, i+1)
		os.Rename(src, dst)
	}

	// Rename current to .1
	os.Rename(base, base+".1")

	// Create new file
	f, err := os.OpenFile(base, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	lw.file = f
	lw.fileSize = 0
}
