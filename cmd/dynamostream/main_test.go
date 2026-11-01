package main

import (
	"bytes"
	"context"
	"log"
	"strings"
	"testing"

	"github.com/aws/aws-lambda-go/events"
)

// captureLogs redirects the standard logger for the duration of the test.
func captureLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	previous := log.Writer()
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(previous) })
	return &buf
}

func streamRecord(eventName string, image map[string]events.DynamoDBAttributeValue) events.DynamoDBEventRecord {
	return events.DynamoDBEventRecord{
		EventName: eventName,
		Change:    events.DynamoDBStreamRecord{NewImage: image},
	}
}

func userImage(id string) map[string]events.DynamoDBAttributeValue {
	return map[string]events.DynamoDBAttributeValue{
		"pk": events.NewStringAttribute("USER#" + id),
		"sk": events.NewStringAttribute("USER"),
		"id": events.NewStringAttribute(id),
	}
}

func TestHandler_LogsNewUsers(t *testing.T) {
	logs := captureLogs(t)

	event := events.DynamoDBEvent{Records: []events.DynamoDBEventRecord{
		streamRecord("INSERT", userImage("user-1")),
		streamRecord("MODIFY", userImage("user-2")),
		streamRecord("REMOVE", nil),
		streamRecord("INSERT", map[string]events.DynamoDBAttributeValue{
			"pk": events.NewStringAttribute("ORDER#1"),
			"sk": events.NewStringAttribute("ORDER"),
			"id": events.NewStringAttribute("order-1"),
		}),
		streamRecord("INSERT", userImage("user-3")),
	}}

	if err := handler(context.Background(), event); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := logs.String()
	for _, want := range []string{"new user created: id=user-1", "new user created: id=user-3"} {
		if !strings.Contains(out, want) {
			t.Errorf("logs missing %q:\n%s", want, out)
		}
	}
	for _, unwanted := range []string{"user-2", "order-1"} {
		if strings.Contains(out, unwanted) {
			t.Errorf("logs should not mention %q:\n%s", unwanted, out)
		}
	}
}

func TestHandler_ToleratesUnexpectedAttributeTypes(t *testing.T) {
	captureLogs(t)

	// A non-string sk would make DynamoDBAttributeValue.String panic if read directly.
	event := events.DynamoDBEvent{Records: []events.DynamoDBEventRecord{
		streamRecord("INSERT", map[string]events.DynamoDBAttributeValue{
			"pk": events.NewStringAttribute("USER#x"),
			"sk": events.NewNumberAttribute("1"),
		}),
		streamRecord("INSERT", map[string]events.DynamoDBAttributeValue{}),
	}}

	if err := handler(context.Background(), event); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStringAttribute(t *testing.T) {
	image := map[string]events.DynamoDBAttributeValue{
		"s": events.NewStringAttribute("value"),
		"n": events.NewNumberAttribute("1"),
	}

	if got := stringAttribute(image, "s"); got != "value" {
		t.Errorf("string attribute = %q, want %q", got, "value")
	}
	if got := stringAttribute(image, "n"); got != "" {
		t.Errorf("number attribute = %q, want empty", got)
	}
	if got := stringAttribute(image, "missing"); got != "" {
		t.Errorf("missing attribute = %q, want empty", got)
	}
}
