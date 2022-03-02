package handler

import (
	"fmt"
	"mime"
	"net/http"
	"strconv"
	"time"

	"nakama/internal/service"
)

func (h *handler) timeline(w http.ResponseWriter, r *http.Request) {
	if a, _, err := mime.ParseMediaType(r.Header.Get("Accept")); err == nil && a == "text/event-stream" {
		h.timelineItemSubscription(w, r)
		return
	}

	ctx := r.Context()
	q := r.URL.Query()
	last, _ := strconv.Atoi(q.Get("last"))
	before, _ := strconv.ParseInt(q.Get("before"), 10, 64)
	tt, err := h.Timeline(ctx, last, before)
	if err == service.ErrUnauthenticated {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	if err != nil {
		respondErr(w, err)
		return
	}

	respond(w, tt, http.StatusOK)
}

func (h *handler) timelineItemSubscription(w http.ResponseWriter, r *http.Request) {
	f, ok := w.(http.Flusher)
	if !ok {
		respondErr(w, errStreamingUnsupported)
		return
	}

	ctx := r.Context()
	tt, err := h.TimelineItemSubscription(ctx)
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
		case ti := <-tt:
			writeSSE(w, ti)
			f.Flush()
		}
	}
}
