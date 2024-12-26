package analysis

import (
	"fmt"
	"log"
	"norminette-lsp/lsp"
	"os"
	"os/exec"
	"strings"
)

type State struct {
	// Map of file names to contents
	Documents map[string]string
}

func NewState() State {
	return State{Documents: map[string]string{}}
}

// Execute norminette and parse its output
func runNorminette(logger *log.Logger, filePath string, text string) ([]lsp.Diagnostic, error) {
	// if we dont provide a text (non saved buffer data) we create a temp file with the text
	// so we can run norminette on it
	if text != "" {
		tempFile, err := os.CreateTemp("/tmp/", "norminette-*.c")
		logger.Println("NewStatefile:", tempFile.Name())
		if err != nil {
			return nil, err
		}
		defer os.Remove(tempFile.Name())

		_, err = tempFile.WriteString(text)
		if err != nil {
			logger.Printf("Failed to write to temp file: %v", err)
			return nil, err
		}
		tempFile.Close()
		filePath = tempFile.Name()
	}
	logger.Println("Running norminette on", filePath)
	fmt.Print("\033[H\033[2J") // ANSI escape code to clear the screen
	cmd := exec.Command("norminette", filePath)
	output, err := cmd.CombinedOutput()
	if err != nil && cmd.ProcessState.ExitCode() != 0 {
		// Norminette returns non-zero exit code for errors; this is not a fatal error.
		// Just process the output.
		fmt.Println("Norminette output:", string(output))
	}

	diagnostics := []lsp.Diagnostic{}
	for _, line := range strings.Split(string(output), "\n") {
		if strings.Contains(line, "Error") {
			// Parse the line to extract error details
			parts := strings.Split(line, ":")
			if len(parts) < 3 {
				continue
			}

			row, col := 0, 0
			fmt.Sscanf(parts[2], "%d", &row)
			fmt.Sscanf(parts[3], "%d", &col)
			message := strings.Join(parts[4:], ":")
			logger.Println("Norminette output line:", line)
			logger.Println("Row:", row)
			logger.Println("Col:", col)
			logger.Println("Message:", message)
			diagnostics = append(diagnostics, lsp.Diagnostic{
				Range:    LineRange(row-1, col-4, col-4),
				Severity: 1, // Error severity
				Source:   "norminette",
				Message:  strings.TrimSpace(message),
			})
		}
	}

	return diagnostics, nil
}

func getDiagnosticsForFile(logger *log.Logger, filePath string, text string) []lsp.Diagnostic {
	logger.Printf("text %s", text)
	diagnostics, err := runNorminette(logger, filePath, text)
	if err != nil {
		fmt.Printf("Failed to run Norminette: %v\n", err)
	}
	return diagnostics
}

func (s *State) OpenDocument(logger *log.Logger, uri, text string) []lsp.Diagnostic {
	// s.Documents[uri] = text
	filePath := strings.TrimPrefix(uri, "file://")

	return getDiagnosticsForFile(logger, filePath, text)
}

func (s *State) SaveDocument(logger *log.Logger, uri, text string) []lsp.Diagnostic {
	// s.Documents[uri] = text
	logger.Println("s.Documents[uri]:", s.Documents)
	filePath := strings.TrimPrefix(uri, "file://")

	return getDiagnosticsForFile(logger, filePath, text)
}

func (s *State) UpdateDocument(logger *log.Logger, uri, text string) []lsp.Diagnostic {
	// s.Documents[uri] = text

	return getDiagnosticsForFile(logger, "", text)
}

// LineRange helper function
func LineRange(line, start, end int) lsp.Range {
	return lsp.Range{
		Start: lsp.Position{
			Line:      line,
			Character: start,
		},
		End: lsp.Position{
			Line:      line,
			Character: end,
		},
	}
}
