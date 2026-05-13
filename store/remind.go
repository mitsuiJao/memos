package store

import (
	"context"

	"github.com/pkg/errors"
	"golang.org/x/crypto/bcrypt"
)

const BotUsername = "reminder_bot"

// Remind represents a pending reminder created via the #remind tag.
type Remind struct {
	ID             int32
	CreatorID      int32
	MemoUID        string
	SequenceNumber int32
	Description    string
	ScheduledTs    int64
	Fired          bool
	Deleted        bool
}

// FindRemind specifies filter criteria for querying reminders.
type FindRemind struct {
	ID             *int32
	CreatorID      *int32
	SequenceNumber *int32
	MemoUID        *string
	// ActiveOnly filters to fired=0 AND deleted=0
	ActiveOnly bool
	// DueOnly implies ActiveOnly and additionally filters scheduled_ts <= NowTs
	DueOnly bool
	NowTs   int64
}

// UpdateRemind carries fields that can be patched on a remind row.
type UpdateRemind struct {
	ID      int32
	Fired   *bool
	Deleted *bool
}

// DeleteRemind identifies a reminder to hard-delete.
type DeleteRemind struct {
	ID int32
}

// RemindEphemeralMemo tracks a bot-created response memo for auto-deletion.
type RemindEphemeralMemo struct {
	MemoUID  string
	ExpireTs int64
}

// FindRemindEphemeralMemo filters ephemeral memo entries.
type FindRemindEphemeralMemo struct {
	// ExpiredBefore filters to expire_ts <= this value (0 = no filter)
	ExpiredBefore int64
}

// CreateRemind creates a new reminder.
func (s *Store) CreateRemind(ctx context.Context, create *Remind) (*Remind, error) {
	return s.driver.CreateRemind(ctx, create)
}

// ListReminds retrieves reminders matching the filter criteria.
func (s *Store) ListReminds(ctx context.Context, find *FindRemind) ([]*Remind, error) {
	return s.driver.ListReminds(ctx, find)
}

// UpdateRemind updates a reminder.
func (s *Store) UpdateRemind(ctx context.Context, update *UpdateRemind) error {
	return s.driver.UpdateRemind(ctx, update)
}

// DeleteRemind hard-deletes a reminder by ID.
func (s *Store) DeleteRemind(ctx context.Context, delete *DeleteRemind) error {
	return s.driver.DeleteRemind(ctx, delete)
}

// CreateRemindEphemeralMemo records a bot response memo for auto-deletion.
func (s *Store) CreateRemindEphemeralMemo(ctx context.Context, create *RemindEphemeralMemo) error {
	return s.driver.CreateRemindEphemeralMemo(ctx, create)
}

// ListRemindEphemeralMemos retrieves ephemeral memo entries matching the filter.
func (s *Store) ListRemindEphemeralMemos(ctx context.Context, find *FindRemindEphemeralMemo) ([]*RemindEphemeralMemo, error) {
	return s.driver.ListRemindEphemeralMemos(ctx, find)
}

// DeleteRemindEphemeralMemo removes an ephemeral memo tracking entry.
func (s *Store) DeleteRemindEphemeralMemo(ctx context.Context, memoUID string) error {
	return s.driver.DeleteRemindEphemeralMemo(ctx, memoUID)
}

// EnsureRemindBotUser looks up or creates the reminder_bot system user.
// Returns the bot user's ID.
func (s *Store) EnsureRemindBotUser(ctx context.Context) (int32, error) {
	username := BotUsername
	users, err := s.ListUsers(ctx, &FindUser{Username: &username})
	if err != nil {
		return 0, errors.Wrap(err, "failed to list users for bot")
	}
	if len(users) > 0 {
		return users[0].ID, nil
	}

	// Bot user doesn't exist yet — create it with a random bcrypt hash (never logs in).
	hash, err := bcrypt.GenerateFromPassword([]byte("reminder-bot-system-account"), bcrypt.DefaultCost)
	if err != nil {
		return 0, errors.Wrap(err, "failed to generate bot password hash")
	}
	user, err := s.CreateUser(ctx, &User{
		Username:     BotUsername,
		Nickname:     "Reminder Bot",
		Role:         RoleUser,
		PasswordHash: string(hash),
	})
	if err != nil {
		return 0, errors.Wrap(err, "failed to create reminder bot user")
	}
	return user.ID, nil
}
