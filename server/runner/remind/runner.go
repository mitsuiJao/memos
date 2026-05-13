package remind

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/lithammer/shortuuid/v4"

	storepb "github.com/usememos/memos/proto/gen/store"
	"github.com/usememos/memos/store"
)

// Runner polls every minute for due reminders and fires notifications.
type Runner struct {
	Store *store.Store
}

// NewRunner creates a new remind runner.
func NewRunner(s *store.Store) *Runner {
	return &Runner{Store: s}
}

const runnerInterval = time.Minute

// Run starts the reminder polling loop.
func (r *Runner) Run(ctx context.Context) {
	ticker := time.NewTicker(runnerInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			r.RunOnce(ctx)
		case <-ctx.Done():
			return
		}
	}
}

// RunOnce fires due reminders and cleans up expired ephemeral memos.
func (r *Runner) RunOnce(ctx context.Context) {
	r.fireReminders(ctx)
	r.cleanEphemeralMemos(ctx)
}

func (r *Runner) fireReminders(ctx context.Context) {
	now := time.Now().UTC()
	reminds, err := r.Store.ListReminds(ctx, &store.FindRemind{
		DueOnly: true,
		NowTs:   now.Unix(),
	})
	if err != nil {
		slog.Error("remind runner: failed to list due reminders", "err", err)
		return
	}

	botID, err := r.Store.EnsureRemindBotUser(ctx)
	if err != nil {
		slog.Error("remind runner: failed to ensure bot user", "err", err)
		return
	}

	for _, remind := range reminds {
		if err := r.fireReminder(ctx, remind, botID); err != nil {
			slog.Error("remind runner: failed to fire reminder", "err", err, "remindID", remind.ID)
			continue
		}
		fired := true
		if err := r.Store.UpdateRemind(ctx, &store.UpdateRemind{ID: remind.ID, Fired: &fired}); err != nil {
			slog.Error("remind runner: failed to mark reminder fired", "err", err, "remindID", remind.ID)
		}
	}
}

func (r *Runner) fireReminder(ctx context.Context, remind *store.Remind, botID int32) error {
	// Look up original memo and creator.
	originalMemo, err := r.Store.GetMemo(ctx, &store.FindMemo{UID: &remind.MemoUID})
	if err != nil {
		return err
	}
	if originalMemo == nil {
		// Original memo was deleted; skip silently.
		return nil
	}

	users, err := r.Store.ListUsers(ctx, &store.FindUser{ID: &remind.CreatorID})
	if err != nil {
		return err
	}
	if len(users) == 0 {
		return fmt.Errorf("creator user %d not found", remind.CreatorID)
	}
	username := users[0].Username

	desc := remind.Description
	if desc == "" {
		desc = "(内容なし)"
	}
	content := fmt.Sprintf("⏰ **リマインダー**\n\n> %s\n\nmemos/%s\n\n@%s",
		desc, remind.MemoUID, username)

	notifMemo, err := r.Store.CreateMemo(ctx, &store.Memo{
		UID:        shortuuid.New(),
		CreatorID:  botID,
		Content:    content,
		Visibility: originalMemo.Visibility,
		ParentUID:  &remind.MemoUID,
	})
	if err != nil {
		return err
	}

	// Link notification memo to original as a comment relation.
	if _, err := r.Store.UpsertMemoRelation(ctx, &store.MemoRelation{
		MemoID:        notifMemo.ID,
		RelatedMemoID: originalMemo.ID,
		Type:          store.MemoRelationComment,
	}); err != nil {
		slog.Warn("remind runner: failed to set memo relation", "err", err)
	}

	// Create inbox notification so the user sees the reminder.
	if _, err := r.Store.CreateInbox(ctx, &store.Inbox{
		SenderID:   botID,
		ReceiverID: remind.CreatorID,
		Status:     store.UNREAD,
		Message: &storepb.InboxMessage{
			Type: storepb.InboxMessage_MEMO_MENTION,
			Payload: &storepb.InboxMessage_MemoMention{
				MemoMention: &storepb.InboxMessage_MemoMentionPayload{
					MemoId:        notifMemo.ID,
					RelatedMemoId: originalMemo.ID,
				},
			},
		},
	}); err != nil {
		slog.Warn("remind runner: failed to create inbox notification", "err", err)
	}

	return nil
}

func (r *Runner) cleanEphemeralMemos(ctx context.Context) {
	now := time.Now().UTC().Unix()
	entries, err := r.Store.ListRemindEphemeralMemos(ctx, &store.FindRemindEphemeralMemo{
		ExpiredBefore: now,
	})
	if err != nil {
		slog.Error("remind runner: failed to list expired ephemeral memos", "err", err)
		return
	}

	for _, entry := range entries {
		memo, err := r.Store.GetMemo(ctx, &store.FindMemo{UID: &entry.MemoUID})
		if err != nil {
			slog.Error("remind runner: failed to get ephemeral memo", "err", err, "uid", entry.MemoUID)
		} else if memo != nil {
			if err := r.Store.DeleteMemo(ctx, &store.DeleteMemo{ID: memo.ID}); err != nil {
				slog.Error("remind runner: failed to delete ephemeral memo", "err", err, "uid", entry.MemoUID)
			}
		}
		if err := r.Store.DeleteRemindEphemeralMemo(ctx, entry.MemoUID); err != nil {
			slog.Error("remind runner: failed to delete ephemeral memo entry", "err", err, "uid", entry.MemoUID)
		}
	}
}
