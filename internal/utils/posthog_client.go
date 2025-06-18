// posthog_client.go provides a wrapper around the posthog.Client to make it easier to use and handle when its not initialized.
package utils

import "github.com/posthog/posthog-go"

// userHandler handles HTTP requests related to users.
type PosthogClientWrapper struct {
	posthogClient posthog.Client
}

func InitializePosthogClient(apiKey string) *PosthogClientWrapper {
	if apiKey == "" {
		return nil
	}
	wrapper := PosthogClientWrapper{}
	wrapper.posthogClient, _ = posthog.NewWithConfig(apiKey, posthog.Config{Endpoint: "https://eu.i.posthog.com"})
	return &wrapper
}

func (w *PosthogClientWrapper) IsInitialized() bool {
	return w.posthogClient != nil
}

func (w *PosthogClientWrapper) Enqueue(distinctId string, event string, properties map[string]any) {
	if w.posthogClient == nil {
		return
	}
	w.posthogClient.Enqueue(posthog.Capture{
		DistinctId: distinctId,
		Event:      event,
		Properties: properties,
	})
}

func (w *PosthogClientWrapper) Close() {
	if w.posthogClient == nil {
		return
	}
	w.posthogClient.Close()
}
