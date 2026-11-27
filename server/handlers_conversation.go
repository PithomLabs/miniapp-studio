package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/PithomLabs/miniapp-studio/tools"
)

// handleConversationsList renders the tenant's conversations plus the
// new-conversation form.
func (s *Server) handleConversationsList(w http.ResponseWriter, r *http.Request) {
	key, err := newOpKey()
	if err != nil {
		renderError(w, r, http.StatusInternalServerError, "", s.logger, err.Error())
		return
	}
	listKey, err := newOpKey()
	if err != nil {
		renderError(w, r, http.StatusInternalServerError, "", s.logger, err.Error())
		return
	}
	res, err := s.invokeTool(r.Context(), r, tools.ToolConversationList,
		map[string]any{"limit": 50}, listKey)
	if err != nil {
		renderError(w, r, http.StatusInternalServerError, "", s.logger, err.Error())
		return
	}
	convs, _ := res["conversations"].([]any)
	if convs == nil {
		convs = []any{}
	}
	s.renderFocusPage(w, r, "scene_conversations.html", "conversations", nil,
		fmt.Sprintf("Development tasks · %d shown", len(convs)), map[string]any{
			"Conversations": convs, "OpKey": key,
		})
}

// handleConversationsCreate opens one conversation and redirects to it.
func (s *Server) handleConversationsCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		renderError(w, r, http.StatusBadRequest, "Bad request.", s.logger, err.Error())
		return
	}
	key, err := opKey(r)
	if err != nil {
		renderError(w, r, http.StatusBadRequest, "Missing operation key.", s.logger, "")
		return
	}
	res, err := s.invokeTool(r.Context(), r, tools.ToolConversationCreate,
		map[string]any{"title": r.Form.Get("title")}, key)
	if err != nil {
		renderError(w, r, http.StatusBadRequest, friendlyToolError(err), s.logger, err.Error())
		return
	}
	id, _ := res["conversation_id"].(string)
	http.Redirect(w, r, "/conversations/"+id, http.StatusFound)
}

// handleConversationView renders one conversation: the persisted stream,
// build statuses with open-question forms, and the message form. Every
// form carries its own fresh operation key — sharing one key across forms
// would collide payload fingerprints.
func (s *Server) handleConversationView(w http.ResponseWriter, r *http.Request) {
	s.renderConversation(w, r, r.PathValue("id"), "")
}

// renderConversation renders the conversation page, optionally reusing one
// submitted message key (failure retry without duplicate sends).
func (s *Server) renderConversation(w http.ResponseWriter, r *http.Request, convID, msgKey string) {
	if convID == "" {
		renderError(w, r, http.StatusNotFound, "Conversation not found.", s.logger, "")
		return
	}
	key, err := newOpKey()
	if err != nil {
		renderError(w, r, http.StatusInternalServerError, "", s.logger, err.Error())
		return
	}
	got, err := s.invokeTool(r.Context(), r, tools.ToolConversationGet,
		map[string]any{"conversation_id": convID}, key)
	if err != nil {
		if isNotFound(err) {
			renderError(w, r, http.StatusNotFound, "Conversation not found.", s.logger, "")
			return
		}
		renderError(w, r, http.StatusInternalServerError, "", s.logger, err.Error())
		return
	}
	streamKey, err := newOpKey()
	if err != nil {
		renderError(w, r, http.StatusInternalServerError, "", s.logger, err.Error())
		return
	}
	page, err := s.invokeTool(r.Context(), r, tools.ToolConversationStream,
		map[string]any{"conversation_id": convID, "after_seq": 0, "limit": 200}, streamKey)
	if err != nil {
		renderError(w, r, http.StatusInternalServerError, "", s.logger, err.Error())
		return
	}
	items, _ := page["items"].([]any)
	latest, _ := page["latest_seq"].(float64)
	if msgKey == "" {
		if msgKey, err = newOpKey(); err != nil {
			renderError(w, r, http.StatusInternalServerError, "", s.logger, err.Error())
			return
		}
	}
	builds, _ := got["builds"].([]any)
	shaped, err := shapeBuilds(builds)
	if err != nil {
		renderError(w, r, http.StatusInternalServerError, "", s.logger, err.Error())
		return
	}
	conversation, _ := got["conversation"].(map[string]any)
	questions := 0
	for _, b := range shaped {
		if m, ok := b.(map[string]any); ok {
			if qs, ok := m["open_questions"].([]any); ok {
				questions += len(qs)
			}
		}
	}
	s.renderFocusPage(w, r, "scene_conversation.html", "conversations", nil,
		fmt.Sprintf("%s · %s", plural(len(shaped), "build"), plural(questions, "open question")), map[string]any{
			// Thread, not Conversation: the shell reserves .Conversation
			// for console focus (it would wrongly enable the Line).
			"Thread": conversation, "Builds": shaped,
			"Items": shapePageItems(items), "LatestSeq": int64(latest),
			"MsgOpKey": msgKey,
		})
}

// shapeBuilds prepares builds for the page: each build keeps its status
// and turn, each open question gains a fresh answer key and parsed options.
func shapeBuilds(builds []any) ([]any, error) {
	out := []any{}
	for _, b := range builds {
		m, ok := b.(map[string]any)
		if !ok {
			continue
		}
		qs, _ := m["open_questions"].([]any)
		shaped := []any{}
		for _, q := range qs {
			qm, ok := q.(map[string]any)
			if !ok {
				continue
			}
			key, err := newOpKey()
			if err != nil {
				return nil, err
			}
			shaped = append(shaped, map[string]any{
				"id": qm["id"], "kind": qm["kind"], "body": qm["body"],
				"options": parseOptions(qm["options"]), "opkey": key,
			})
		}
		m["open_questions"] = shaped
		out = append(out, m)
	}
	return out, nil
}

// parseOptions decodes a question's options for the template. The tool
// already scrubbed and decoded them; raw bytes survive only as a
// defensive path. Corrupt data renders as no options, never an error.
func parseOptions(v any) []any {
	switch t := v.(type) {
	case []any:
		if t == nil {
			return []any{}
		}
		return t
	case []byte:
		return parseOptionsBytes(t)
	case string:
		return parseOptionsBytes([]byte(t))
	default:
		return []any{}
	}
}

func parseOptionsBytes(raw []byte) []any {
	var opts []any
	if err := json.Unmarshal(raw, &opts); err != nil {
		return []any{}
	}
	if opts == nil {
		return []any{}
	}
	return opts
}

// pageItem is one stream row shaped for server rendering.
type pageItem struct {
	Seq  int64
	Kind string
	Role string
	Body string
	Type string
	Text string
}

// shapePageItems renders stream rows for the template: messages show
// role + body; events show their type plus a short extracted text.
func shapePageItems(items []any) []pageItem {
	out := []pageItem{}
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		seq, _ := m["seq"].(float64)
		kind, _ := m["kind"].(string)
		pi := pageItem{Seq: int64(seq), Kind: kind}
		if kind == "message" {
			pi.Role, _ = m["role"].(string)
			pi.Body, _ = m["body"].(string)
		} else {
			pi.Type, _ = m["type"].(string)
			pi.Text = eventText(m["type"], m["payload"])
		}
		out = append(out, pi)
	}
	return out
}

// eventText extracts one display line from an event payload. Unknown shapes
// fall back to the event type so nothing renders blank.
func eventText(typ, payload any) string {
	name, _ := typ.(string)
	var raw []byte
	switch t := payload.(type) {
	case []byte:
		raw = t
	case string:
		raw = []byte(t)
	case map[string]any:
		for _, k := range []string{"text", "body"} {
			if s, _ := t[k].(string); s != "" {
				return s
			}
		}
		return name
	default:
		return name
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return name
	}
	for _, k := range []string{"text", "body"} {
		if s, _ := m[k].(string); s != "" {
			return s
		}
	}
	return name
}

// handleMessageSend appends one user message (atomically ensuring a build)
// and redirects back to the conversation.
func (s *Server) handleMessageSend(w http.ResponseWriter, r *http.Request) {
	convID := r.PathValue("id")
	if err := r.ParseForm(); err != nil {
		renderError(w, r, http.StatusBadRequest, "Bad request.", s.logger, err.Error())
		return
	}
	key, err := opKey(r)
	if err != nil {
		renderError(w, r, http.StatusBadRequest, "Missing operation key.", s.logger, "")
		return
	}
	if r.Form.Get("body") == "" {
		renderError(w, r, http.StatusBadRequest, "Message body required.", s.logger, "")
		return
	}
	_, err = s.invokeTool(r.Context(), r, tools.ToolMessageSend,
		map[string]any{"conversation_id": convID, "body": r.Form.Get("body")}, key)
	if err != nil {
		if isNotFound(err) {
			renderError(w, r, http.StatusNotFound, "Conversation not found.", s.logger, "")
			return
		}
		renderError(w, r, http.StatusBadRequest, friendlyToolError(err), s.logger, err.Error())
		return
	}
	http.Redirect(w, r, "/conversations/"+convID, http.StatusFound)
}

// handleQuestionAnswer records one answer. Answering a stale question
// renders 409 with the current open question instead of applying silently.
func (s *Server) handleQuestionAnswer(w http.ResponseWriter, r *http.Request) {
	qid := r.PathValue("qid")
	if err := r.ParseForm(); err != nil {
		renderError(w, r, http.StatusBadRequest, "Bad request.", s.logger, err.Error())
		return
	}
	key, err := opKey(r)
	if err != nil {
		renderError(w, r, http.StatusBadRequest, "Missing operation key.", s.logger, "")
		return
	}
	buildID := r.Form.Get("build_id")
	res, err := s.invokeTool(r.Context(), r, tools.ToolQuestionAnswer,
		map[string]any{"question_id": qid, "answer": r.Form.Get("answer")}, key)
	if err != nil {
		if isStaleQuestion(err) && buildID != "" {
			s.renderAnswerConflict(w, r, buildID)
			return
		}
		if isNotFound(err) {
			renderError(w, r, http.StatusNotFound, "Question not found.", s.logger, "")
			return
		}
		renderError(w, r, http.StatusBadRequest, friendlyToolError(err), s.logger, err.Error())
		return
	}
	// The answer carries its build; resolve the conversation for PRG.
	stKey, kerr := newOpKey()
	if kerr != nil {
		renderError(w, r, http.StatusInternalServerError, "", s.logger, kerr.Error())
		return
	}
	st, serr := s.invokeTool(r.Context(), r, tools.ToolBuildStatus,
		map[string]any{"build_id": res["build_id"]}, stKey)
	if serr != nil {
		renderError(w, r, http.StatusInternalServerError, "", s.logger, serr.Error())
		return
	}
	convID, _ := st["conversation_id"].(string)
	http.Redirect(w, r, "/conversations/"+convID, http.StatusFound)
}

func isStaleQuestion(err error) bool {
	return err != nil && strings.Contains(err.Error(), "no longer open")
}

// renderAnswerConflict shows the 409 page: the asked question went stale,
// and this is the current open question for the build (if any).
func (s *Server) renderAnswerConflict(w http.ResponseWriter, r *http.Request, buildID string) {
	key, err := newOpKey()
	if err != nil {
		renderError(w, r, http.StatusInternalServerError, "", s.logger, err.Error())
		return
	}
	st, err := s.invokeTool(r.Context(), r, tools.ToolBuildStatus,
		map[string]any{"build_id": buildID}, key)
	if err != nil {
		renderError(w, r, http.StatusInternalServerError, "", s.logger, err.Error())
		return
	}
	convID, _ := st["conversation_id"].(string)
	w.WriteHeader(http.StatusConflict)
	s.renderFocusPage(w, r, "scene_answer_conflict.html", "conversations", nil,
		"Stale answer rejected", map[string]any{
			"BuildID": buildID, "ConversationID": convID,
			"OpenQuestions": st["open_questions"],
		})
}
