package handler

import (
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"strconv"
	"time"

	"nakama/internal/service"

	"github.com/matryer/way"
)

type createCommentInput struct {
	Content string
}

func (h *handler) createComment(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var in createCommentInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	postID, _ := strconv.ParseInt(way.Param(ctx, "post_id"), 10, 64)
	c, err := h.CreateComment(ctx, postID, in.Content)
	if err == service.ErrUnauthenticated {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	if err == service.ErrInvalidContent {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	if err == service.ErrPostNotFound {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	if err != nil {
		respondErr(w, err)
		return
	}

	respond(w, c, http.StatusCreated)
}

func (h *handler) comments(w http.ResponseWriter, r *http.Request) {
	if a, _, err := mime.ParseMediaType(r.Header.Get("Accept")); err == nil && a == "text/event-stream" {
		h.commentSubscription(w, r)
		return
	}

	ctx := r.Context()
	q := r.URL.Query()
	postID, _ := strconv.ParseInt(way.Param(ctx, "post_id"), 10, 64)
	last, _ := strconv.Atoi(q.Get("last"))
	before, _ := strconv.ParseInt(q.Get("before"), 10, 64)
	cc, err := h.Comments(ctx, postID, last, before)
	if err != nil {
		respondErr(w, err)
		return
	}

	respond(w, cc, http.StatusOK)
}

func (h *handler) commentSubscription(w http.ResponseWriter, r *http.Request) {
	f, ok := w.(http.Flusher)
	if !ok {
		respondErr(w, errStreamingUnsupported)
		return
	}

	ctx := r.Context()
	postID, _ := strconv.ParseInt(way.Param(ctx, "post_id"), 10, 64)

	header := w.Header()
	header.Set("Cache-Control", "no-cache")
	header.Set("Connection", "keep-alive")
	header.Set("Content-Type", "text/event-stream")

	cc := h.CommentSubscription(ctx, postID)

	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(h.ping):
			fmt.Fprint(w, "ping: \n\n")
			f.Flush()
		case c := <-cc:
			writeSSE(w, c)
			f.Flush()
		}
	}
}

func (h *handler) toggleCommentLike(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	commentID, _ := strconv.ParseInt(way.Param(ctx, "comment_id"), 10, 64)
	out, err := h.ToggleCommentLike(ctx, commentID)
	if err == service.ErrUnauthenticated {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	if err == service.ErrCommentNotFound {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	if err != nil {
		respondErr(w, err)
		return
	}

	respond(w, out, http.StatusOK)
}
