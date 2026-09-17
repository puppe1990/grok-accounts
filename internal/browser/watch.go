package browser

import (
	"bufio"
	"errors"
	"io"
	"strings"
)

func WatchURLs(r io.Reader, dst io.Writer, open func(string)) error {
	reader := bufio.NewReader(r)
	found := false

	for {
		line, err := reader.ReadString('\n')
		if line != "" {
			if _, writeErr := io.WriteString(dst, line); writeErr != nil {
				return writeErr
			}
			if !found {
				if url, ok := firstURL(line); ok {
					found = true
					open(url)
				}
			}
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
	}
}

func firstURL(line string) (string, bool) {
	for _, field := range strings.Fields(line) {
		if strings.HasPrefix(field, "https://") || strings.HasPrefix(field, "http://") {
			return strings.TrimRight(field, ".,;:!?"), true
		}
	}
	return "", false
}
