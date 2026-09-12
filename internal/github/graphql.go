package github

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// graphql runs one GraphQL document and unmarshals its `data` into out.
// GitHub answers a rejected query with HTTP 200 and an `errors` array, so the
// envelope has to be inspected even on success.
func (c *Client) graphql(ctx context.Context, query string, variables map[string]any, out any) error {
	body, err := json.Marshal(map[string]any{"query": query, "variables": variables})
	if err != nil {
		return fmt.Errorf("encode GraphQL request: %w", err)
	}

	var envelope struct {
		Data   json.RawMessage `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := c.send(ctx, "POST", c.endpoint(nil, "graphql"), body, &envelope); err != nil {
		return err
	}
	if len(envelope.Errors) > 0 {
		messages := make([]string, len(envelope.Errors))
		for i, e := range envelope.Errors {
			messages[i] = e.Message
		}
		return &UpstreamError{
			Summary: "GitHub GraphQL error",
			Detail:  strings.Join(messages, "; "),
		}
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(envelope.Data, out); err != nil {
		return &UpstreamError{Summary: "GitHub sent an unreadable response", Detail: err.Error()}
	}
	return nil
}
