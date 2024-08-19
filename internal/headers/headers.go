package headers

import (
	"bytes"
	"fmt"
	"github.com/emersion/go-message"
	_ "github.com/emersion/go-message/charset"
	"mime"
	"strings"
)

type Header struct {
	Key   string
	Value string
}

func Parse(emlContent []byte) ([]Header, error) {
	msg, err := message.Read(bytes.NewReader(emlContent))
	if err != nil {
		return nil, fmt.Errorf("failed to read message: %v", err)
	}

	var headers []Header
	fields := msg.Header.Fields()
	for fields.Next() {
		key := fields.Key()
		rawValue := fields.Value()

		decodedValue, err := (&mime.WordDecoder{}).DecodeHeader(rawValue)
		if err != nil {
			decodedValue = rawValue // Use raw value as a fallback
		}

		headers = append(headers, Header{Key: key, Value: decodedValue})
	}

	return headers, nil
}

func Format(headers []Header) string {
	var headerBuilder strings.Builder
	for _, header := range headers {
		headerLine := fmt.Sprintf("%s: %s", header.Key, header.Value)
		lines := splitLongLines(headerLine, 76)

		for i, ln := range lines {
			if i == 0 {
				headerBuilder.WriteString(ln + "\n")
			} else {
				headerBuilder.WriteString("\t" + ln + "\n")
			}
		}
	}
	return headerBuilder.String()
}

func FormatSelected(headers []Header, targetHeaders []string) string {
	targetHeadersMap := make(map[string]struct{})
	for _, key := range targetHeaders {
		targetHeadersMap[key] = struct{}{}
	}

	var headerBuilder strings.Builder

	for _, header := range headers {
		if _, wanted := targetHeadersMap[header.Key]; wanted {
			headerBuilder.WriteString(fmt.Sprintf("%s: %s\n", header.Key, header.Value))
		}
	}

	return headerBuilder.String()
}

func splitLongLines(s string, maxLength int) []string {
	var lines []string
	for len(s) > maxLength {
		idx := strings.LastIndexAny(s[:maxLength], " \t")
		if idx == -1 || idx == 0 {
			// If no space/tab found or it's at the beginning,
			// force a split at maxLength
			idx = maxLength
		}
		lines = append(lines, s[:idx])
		s = strings.TrimSpace(s[idx:])
	}
	if len(s) > 0 {
		lines = append(lines, s)
	}
	return lines
}
