package hermes

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Events returns evidence in transport order. It deliberately does not
// deduplicate or reorder IDs; lifecycle compare-and-set belongs to the Host.
// EOF is reported as a disconnect because a Runs event stream is expected to
// remain open while work is active. Events parsed before EOF are returned too.
func (c *Client) Events(ctx context.Context, runID string) ([]Event, error) {
	path, err := runPath(runID, "/events")
	if err != nil {
		return nil, err
	}
	u := *c.baseURL
	u.Path = strings.TrimRight(u.Path, "/") + path
	u.RawPath = ""
	token, err := c.token()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, c.eventTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "text/event-stream")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, readErr := boundedRead(resp.Body, c.maxResponseBytes, ErrResponseTooLarge)
		if readErr != nil {
			return nil, readErr
		}
		return nil, safeHTTPError(resp.StatusCode, b)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "" && !strings.HasPrefix(strings.ToLower(ct), "text/event-stream") {
		return nil, fmt.Errorf("unexpected Hermes events content type %q", ct)
	}
	return parseSSEBounded(resp.Body, c.maxEventBytes, c.maxEvents, c.maxEventStreamBytes)
}

func parseSSE(r io.Reader, maxEventBytes int64) ([]Event, error) {
	return parseSSEBounded(r, maxEventBytes, defaultMaxEvents, maxEventBytes)
}

func parseSSEBounded(r io.Reader, maxEventBytes int64, maxEvents int, maxStreamBytes int64) ([]Event, error) {
	if maxEventBytes <= 0 || maxEvents <= 0 || maxStreamBytes <= 0 || maxStreamBytes >= 1<<63-1 {
		return nil, ErrEventTooLarge
	}
	scanner := bufio.NewScanner(r)
	maxToken := maxEventBytes + 2
	if maxToken < maxEventBytes || maxToken > int64(int(^uint(0)>>1)) {
		return nil, ErrEventTooLarge
	}
	scanner.Buffer(make([]byte, 1024), int(maxToken))
	var (
		events        []Event
		id, typ, data string
		size          int64
		streamSize    int64
		hasField      bool
		retry         int
	)
	flush := func() error {
		if !hasField {
			return nil
		}
		if data == "" {
			id, typ, size, hasField, retry = "", "", 0, false, 0
			return nil
		}
		if !json.Valid([]byte(data)) {
			return errors.New("invalid Hermes SSE JSON data")
		}
		if len(events) >= maxEvents {
			return ErrEventStreamTooLarge
		}
		events = append(events, Event{ID: id, Type: typ, Retry: retry, Data: json.RawMessage(data)})
		id, typ, data, size, hasField, retry = "", "", "", 0, false, 0
		return nil
	}
	for scanner.Scan() {
		line := scanner.Bytes()
		if int64(len(line))+size+1 > maxEventBytes {
			return events, ErrEventTooLarge
		}
		if streamSize+int64(len(line))+1 > maxStreamBytes {
			return events, ErrEventStreamTooLarge
		}
		streamSize += int64(len(line)) + 1
		size += int64(len(line)) + 1
		if len(line) == 0 {
			if err := flush(); err != nil {
				return events, err
			}
			continue
		}
		if line[0] == ':' {
			continue
		}
		field, value, found := strings.Cut(string(line), ":")
		if found && strings.HasPrefix(value, " ") {
			value = value[1:]
		}
		if !found {
			field, value = string(line), ""
		}
		hasField = true
		switch field {
		case "id":
			id = value
		case "event":
			typ = value
		case "data":
			if data != "" {
				data += "\n"
			}
			data += value
			if int64(len(data)) > maxEventBytes {
				return events, ErrEventTooLarge
			}
		case "retry":
			parsed, err := parseRetry(value)
			if err != nil {
				return events, err
			}
			retry = parsed
		default:
			return events, fmt.Errorf("unsupported Hermes SSE field %q", field)
		}
	}
	if err := scanner.Err(); err != nil {
		if errors.Is(err, bufio.ErrTooLong) {
			return events, ErrEventTooLarge
		}
		return events, err
	}
	// A record without its blank-line delimiter is not evidence. Preserve the
	// disconnect result, but discard the incomplete fields.
	return events, ErrEventStreamDisconnected
}

func parseRetry(value string) (int, error) {
	if value == "" {
		return 0, errors.New("invalid Hermes SSE retry value")
	}
	for i := 0; i < len(value); i++ {
		if value[i] < '0' || value[i] > '9' {
			return 0, errors.New("invalid Hermes SSE retry value")
		}
	}
	var retry int
	for i := 0; i < len(value); i++ {
		if retry > (int(^uint(0)>>1)-int(value[i]-'0'))/10 {
			return 0, errors.New("invalid Hermes SSE retry value")
		}
		retry = retry*10 + int(value[i]-'0')
	}
	return retry, nil
}
