package webhook

import (
	"context"
	"net/http"

	"github.com/google/go-github/v45/github"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// WebhookHandler handles GitHub webhooks
type WebhookHandler struct {
	client.Client
	webhookSecret []byte
}

// NewWebhookHandler creates a new webhook handler
func NewWebhookHandler(client client.Client, secret []byte) *WebhookHandler {
	return &WebhookHandler{
		Client:        client,
		webhookSecret: secret,
	}
}

func (h *WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logger := log.FromContext(r.Context())

	// Validate webhook payload
	payload, err := github.ValidatePayload(r, h.webhookSecret)
	if err != nil {
		logger.Error(err, "invalid webhook payload")
		http.Error(w, "Invalid webhook payload", http.StatusBadRequest)
		return
	}

	// Parse the webhook event
	event, err := github.ParseWebHook(github.WebHookType(r), payload)
	if err != nil {
		logger.Error(err, "failed to parse webhook")
		http.Error(w, "Failed to parse webhook", http.StatusBadRequest)
		return
	}

	// Handle different event types
	switch e := event.(type) {
	case *github.RepositoryEvent:
		if err := h.handleRepositoryEvent(r.Context(), e); err != nil {
			logger.Error(err, "failed to handle repository event")
			http.Error(w, "Failed to handle repository event", http.StatusInternalServerError)
			return
		}
	case *github.PushEvent:
		if err := h.handlePushEvent(r.Context(), e); err != nil {
			logger.Error(err, "failed to handle push event")
			http.Error(w, "Failed to handle push event", http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
}

func (h *WebhookHandler) handleRepositoryEvent(ctx context.Context, event *github.RepositoryEvent) error {
	logger := log.FromContext(ctx)
	logger.Info("handling repository event",
		"action", event.GetAction(),
		"repository", event.GetRepo().GetFullName())

	// Trigger reconciliation for affected repository
	// Implementation depends on your specific requirements
	return nil
}

func (h *WebhookHandler) handlePushEvent(ctx context.Context, event *github.PushEvent) error {
	logger := log.FromContext(ctx)
	logger.Info("handling push event",
		"repository", event.GetRepo().GetFullName(),
		"ref", event.GetRef())

	// Trigger reconciliation for affected repository
	// Implementation depends on your specific requirements
	return nil
}
