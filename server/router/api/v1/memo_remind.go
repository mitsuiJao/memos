package v1

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/lithammer/shortuuid/v4"

	remindrunner "github.com/usememos/memos/server/runner/remind"
	"github.com/usememos/memos/store"
)

const remindEphemeralDuration = 30 * time.Minute

// processRemindCommand is called after CreateMemo when the memo contains a #remind tag.
// It runs in a separate goroutine so it never blocks the API response.
func (s *APIV1Service) processRemindCommand(ctx context.Context, user *store.User, memo *store.Memo) {
	cmd := remindrunner.ParseRemindContent(memo.Content, time.Now().UTC())
	if cmd == nil {
		return
	}

	botID, err := s.Store.EnsureRemindBotUser(ctx)
	if err != nil {
		slog.Error("remind: failed to ensure bot user", "err", err)
		return
	}

	switch {
	case cmd.ParseError != nil:
		s.createRemindErrorMemo(ctx, botID, memo, cmd.ParseError.Error())
	case cmd.IsListCommand:
		s.handleRemindList(ctx, user, memo, botID)
	case cmd.IsDelCommand:
		s.handleRemindDel(ctx, user, memo, botID, cmd.DisplayNumber)
	case cmd.IsSchedule:
		s.scheduleRemind(ctx, user, memo, botID, cmd)
	}
}

// scheduleRemind persists a new reminder and posts a confirmation ephemeral memo.
func (s *APIV1Service) scheduleRemind(ctx context.Context, user *store.User, memo *store.Memo, botID int32, cmd *remindrunner.RemindCommand) {
	remind, err := s.Store.CreateRemind(ctx, &store.Remind{
		CreatorID:   user.ID,
		MemoUID:     memo.UID,
		Description: cmd.Description,
		ScheduledTs: cmd.ScheduledTime.Unix(),
	})
	if err != nil {
		slog.Error("remind: failed to create remind", "err", err)
		s.createRemindErrorMemo(ctx, botID, memo, "リマインダーの登録に失敗しました")
		return
	}

	desc := cmd.Description
	if desc == "" {
		desc = "(内容なし)"
	}
	timeStr := cmd.ScheduledTime.UTC().Format("2006/01/02 15:04 UTC")
	content := fmt.Sprintf("✅ **リマインダー登録完了**\n\n**内容**: %s  \n**時刻**: %s  \n**番号**: No.%d\n\n@%s",
		desc, timeStr, remind.SequenceNumber, user.Username)

	s.createRemindEphemeralMemo(ctx, botID, memo, content)
}

// handleRemindList returns the list of active reminders as an ephemeral memo.
func (s *APIV1Service) handleRemindList(ctx context.Context, user *store.User, memo *store.Memo, botID int32) {
	reminds, err := s.Store.ListReminds(ctx, &store.FindRemind{
		CreatorID:  &user.ID,
		ActiveOnly: true,
	})
	if err != nil {
		slog.Error("remind: failed to list reminds", "err", err)
		s.createRemindErrorMemo(ctx, botID, memo, "リマインダー一覧の取得に失敗しました")
		return
	}

	var content string
	if len(reminds) == 0 {
		content = fmt.Sprintf("📋 **リマインド一覧**\n\nリマインドは登録されていません。\n\n@%s", user.Username)
	} else {
		content = "📋 **リマインド一覧**\n\n"
		for i, r := range reminds {
			desc := r.Description
			if desc == "" {
				desc = "(内容なし)"
			}
			timeStr := time.Unix(r.ScheduledTs, 0).UTC().Format("01/02 15:04 UTC")
			content += fmt.Sprintf("%d. **%s** → %s\n", i+1, desc, timeStr)
		}
		content += fmt.Sprintf("\n削除するには `#remind del <番号>` を投稿してください。\n\n@%s", user.Username)
	}

	s.createRemindEphemeralMemo(ctx, botID, memo, content)
}

// handleRemindDel soft-deletes the Nth active reminder (1-indexed display order).
func (s *APIV1Service) handleRemindDel(ctx context.Context, user *store.User, memo *store.Memo, botID int32, displayNum int) {
	reminds, err := s.Store.ListReminds(ctx, &store.FindRemind{
		CreatorID:  &user.ID,
		ActiveOnly: true,
	})
	if err != nil {
		slog.Error("remind: failed to list reminds for del", "err", err)
		s.createRemindErrorMemo(ctx, botID, memo, "リマインダー一覧の取得に失敗しました")
		return
	}

	if displayNum < 1 || displayNum > len(reminds) {
		s.createRemindErrorMemo(ctx, botID, memo,
			fmt.Sprintf("No.%d は存在しません。`#remind list` で一覧を確認してください。", displayNum))
		return
	}

	target := reminds[displayNum-1]
	deleted := true
	if err := s.Store.UpdateRemind(ctx, &store.UpdateRemind{ID: target.ID, Deleted: &deleted}); err != nil {
		slog.Error("remind: failed to delete remind", "err", err)
		s.createRemindErrorMemo(ctx, botID, memo, "リマインダーの削除に失敗しました")
		return
	}

	desc := target.Description
	if desc == "" {
		desc = "(内容なし)"
	}
	content := fmt.Sprintf("🗑️ **削除完了**\n\n> No.%d %s\n\n@%s", displayNum, desc, user.Username)
	s.createRemindEphemeralMemo(ctx, botID, memo, content)
}

// createRemindErrorMemo posts an ephemeral error response as a comment on the original memo.
func (s *APIV1Service) createRemindErrorMemo(ctx context.Context, botID int32, parentMemo *store.Memo, errMsg string) {
	// Look up original creator's username for the @mention.
	username := ""
	users, err := s.Store.ListUsers(ctx, &store.FindUser{ID: &parentMemo.CreatorID})
	if err == nil && len(users) > 0 {
		username = users[0].Username
	}

	content := fmt.Sprintf("❌ **#remind エラー**\n\n> %s\n\n**使い方:**\n- `#remind 1130 洗濯` — 今日 11:30 にリマインド\n- `2200 買い物 #remind` — 今日 22:00 にリマインド\n- `#remind 05141130 買い物` — 5/14 11:30 にリマインド\n- `#remind list` — 一覧を表示\n- `#remind del 1` — No.1 を削除\n\n@%s",
		errMsg, username)
	s.createRemindEphemeralMemo(ctx, botID, parentMemo, content)
}

// createRemindEphemeralMemo creates a bot response memo as a comment on parentMemo
// and registers it for auto-deletion after 30 minutes.
func (s *APIV1Service) createRemindEphemeralMemo(ctx context.Context, botID int32, parentMemo *store.Memo, content string) {
	uid := shortuuid.New()
	responseMemo, err := s.Store.CreateMemo(ctx, &store.Memo{
		UID:        uid,
		CreatorID:  botID,
		Content:    content,
		Visibility: parentMemo.Visibility,
		ParentUID:  &parentMemo.UID,
	})
	if err != nil {
		slog.Error("remind: failed to create ephemeral memo", "err", err)
		return
	}

	if _, err := s.Store.UpsertMemoRelation(ctx, &store.MemoRelation{
		MemoID:        responseMemo.ID,
		RelatedMemoID: parentMemo.ID,
		Type:          store.MemoRelationComment,
	}); err != nil {
		slog.Warn("remind: failed to set comment relation", "err", err)
	}

	expireTs := time.Now().UTC().Add(remindEphemeralDuration).Unix()
	if err := s.Store.CreateRemindEphemeralMemo(ctx, &store.RemindEphemeralMemo{
		MemoUID:  uid,
		ExpireTs: expireTs,
	}); err != nil {
		slog.Warn("remind: failed to register ephemeral memo", "err", err)
	}
}

// cleanupMemoReminds soft-deletes all active reminders associated with a deleted memo.
func (s *APIV1Service) cleanupMemoReminds(ctx context.Context, memoUID string) {
	reminds, err := s.Store.ListReminds(ctx, &store.FindRemind{
		MemoUID:    &memoUID,
		ActiveOnly: true,
	})
	if err != nil {
		slog.Error("remind: failed to list reminds for cleanup", "err", err)
		return
	}

	deleted := true
	for _, r := range reminds {
		if err := s.Store.UpdateRemind(ctx, &store.UpdateRemind{ID: r.ID, Deleted: &deleted}); err != nil {
			slog.Error("remind: failed to soft-delete remind on memo delete", "err", err, "remindID", r.ID)
		}
	}
}
