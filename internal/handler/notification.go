package handler

import (
	"fmt"
	"mime"
	"net/http"
	"strconv"
	"time"

	"nakama/internal/service"

	"github.com/matryer/way"
)

func (h *handler) notifications(w http.ResponseWriter, r *http.Request) {
	if a, _, err := mime.ParseMediaType(r.Header.Get("Accept")); err == nil && a == "text/event-stream" {
		h.notificationSubscription(w, r)
		return
	}

	q := r.URL.Query()
	last, _ := strconv.Atoi(q.Get("last"))
	before, _ := strconv.ParseInt(q.Get("before"), 10, 64)
	nn, err := h.Notifications(r.Context(), last, before)
	if err == service.ErrUnauthenticated {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	if err != nil {
		respondErr(w, err)
		return
	}

	respond(w, nn, http.StatusOK)
}

func (h *handler) notificationSubscription(w http.ResponseWriter, r *http.Request) {
	f, ok := w.(http.Flusher)
	if !ok {
		respondErr(w, errStreamingUnsupported)
		return
	}

	ctx := r.Context()
	nn, err := h.NotificationSubscription(ctx)
	if err == service.ErrUnauthenticated {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	if err != nil {
		respondErr(w, err)
		return
	}

	header := w.Header()
	header.Set("Cache-Control", "no-cache")
	header.Set("Connection", "keep-alive")
	header.Set("Content-Type", "text/event-stream")

	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(h.ping):
			fmt.Fprint(w, "ping: \n\n")
			f.Flush()
		case n := <-nn:
			writeSSE(w, n)
			f.Flush()
		}
	}
}

func (h *handler) hasUnreadNotifications(w http.ResponseWriter, r *http.Request) {
	unread, err := h.HasUnreadNotifications(r.Context())
	if err == service.ErrUnauthenticated {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	if err != nil {
		respondErr(w, err)
		return
	}

	respond(w, unread, http.StatusOK)
}

func (h *handler) markNotificationAsRead(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	notificationID, _ := strconv.ParseInt(way.Param(ctx, "notification_id"), 10, 64)
	err := h.MarkNotificationAsRead(ctx, notificationID)
	if err == service.ErrUnauthenticated {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	if err != nil {
		respondErr(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) markNotificationsAsRead(w http.ResponseWriter, r *http.Request) {
	err := h.MarkNotificationsAsRead(r.Context())
	if err == service.ErrUnauthenticated {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	if err != nil {
		respondErr(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
