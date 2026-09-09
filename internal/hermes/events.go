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

// EventIterator exposes bounded SSE records without waiting for the stream to
// close. The Host owns lifecycle and deduplication; the adapter only parses.
type EventIterator interface {
	Next(context.Context) (Event, error)
	Close() error
}

type StreamAdapter interface {
	EventsStream(context.Context, string) (EventIterator, error)
}

// Events returns evidence in transport order. It deliberately does not
// deduplicate or reorder IDs; lifecycle compare-and-set belongs to the Host.
// EOF is reported as a disconnect because a Runs event stream is expected to
// remain open while work is active. Events parsed before EOF are returned too.
func (c *Client) Events(ctx context.Context, runID string) ([]Event, error) {
	stream, err := c.EventsStream(ctx, runID)
	if err != nil {
		return nil, err
	}
	defer stream.Close()
	var events []Event
	for {
		event, nextErr := stream.Next(ctx)
		if nextErr != nil {
			return events, nextErr
		}
		events = append(events, event)
	}
}

// EventsStream opens a bounded, incremental SSE reader. It returns as soon as
// the first complete record arrives and keeps the HTTP response open for the
// next record, allowing approval requests to be handled interactively.
func (c *Client) EventsStream(ctx context.Context, runID string) (EventIterator, error) {
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
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		cancel()
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "text/event-stream")
	resp, err := c.http.Do(req)
	if err != nil {
		cancel()
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, readErr := boundedRead(resp.Body, c.maxResponseBytes, ErrResponseTooLarge)
		if readErr != nil {
			_ = resp.Body.Close()
			cancel()
			return nil, readErr
		}
		_ = resp.Body.Close()
		cancel()
		return nil, safeHTTPError(resp.StatusCode, b)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "" && !strings.HasPrefix(strings.ToLower(ct), "text/event-stream") {
		_ = resp.Body.Close()
		cancel()
		return nil, fmt.Errorf("unexpected Hermes events content type %q", ct)
	}
	return &sseIterator{body: resp.Body, cancel: cancel, scanner: newSSEScanner(resp.Body, c.maxEventBytes), maxEventBytes: c.maxEventBytes, maxEvents: c.maxEvents, maxStreamBytes: c.maxEventStreamBytes}, nil
}

type sseIterator struct {
	body           io.ReadCloser
	cancel         context.CancelFunc
	scanner        *bufio.Scanner
	maxEventBytes  int64
	maxEvents      int
	maxStreamBytes int64
	eventCount     int
	streamSize     int64
	id, typ, data  string
	retry          int
	hasField       bool
	eventSize      int64
}

func newSSEScanner(r io.Reader, max int64) *bufio.Scanner {
	scanner := bufio.NewScanner(r)
	maxToken := max + 2
	if maxToken < max || maxToken > int64(int(^uint(0)>>1)) {
		maxToken = int64(int(^uint(0) >> 1))
	}
	scanner.Buffer(make([]byte, 1024), int(maxToken))
	return scanner
}

func (s *sseIterator) Next(_ context.Context) (Event, error) {
	for s.scanner.Scan() {
		line := s.scanner.Bytes()
		if int64(len(line))+s.eventSize+1 > s.maxEventBytes || s.streamSize+int64(len(line))+1 > s.maxStreamBytes {
			return Event{}, ErrEventStreamTooLarge
		}
		s.eventSize += int64(len(line)) + 1
		s.streamSize += int64(len(line)) + 1
		if len(line) == 0 {
			if !s.hasField {
				continue
			}
			if s.data == "" {
				s.reset()
				continue
			}
			if !json.Valid([]byte(s.data)) {
				return Event{}, errors.New("invalid Hermes SSE JSON data")
			}
			if s.eventCount >= s.maxEvents {
				return Event{}, ErrEventStreamTooLarge
			}
			event := Event{ID: s.id, Type: s.typ, Retry: s.retry, Data: json.RawMessage(s.data)}
			s.eventCount++
			s.reset()
			return event, nil
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
		s.hasField = true
		switch field {
		case "id":
			s.id = value
		case "event":
			s.typ = value
		case "data":
			if s.data != "" {
				s.data += "\n"
			}
			s.data += value
		case "retry":
			parsed, err := parseRetry(value)
			if err != nil {
				return Event{}, err
			}
			s.retry = parsed
		default:
			return Event{}, fmt.Errorf("unsupported Hermes SSE field %q", field)
		}
	}
	if err := s.scanner.Err(); err != nil {
		if errors.Is(err, bufio.ErrTooLong) {
			return Event{}, ErrEventTooLarge
		}
		return Event{}, err
	}
	return Event{}, ErrEventStreamDisconnected
}

func (s *sseIterator) reset() {
	s.id, s.typ, s.data, s.retry, s.hasField, s.eventSize = "", "", "", 0, false, 0
}

func (s *sseIterator) Close() error {
	s.cancel()
	return s.body.Close()
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
