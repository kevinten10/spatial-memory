package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spatial-memory/spatial-memory/internal/model"
)

// CircleRepository handles friend circle operations.
type CircleRepository interface {
	Create(ctx context.Context, circle *model.FriendCircle) error
	GetByID(ctx context.Context, id int64) (*model.FriendCircle, error)
	ListByOwner(ctx context.Context, ownerID int64, page, pageSize int) ([]*model.FriendCircle, error)
	ListByMember(ctx context.Context, userID int64, page, pageSize int) ([]*model.FriendCircle, error)
	Update(ctx context.Context, circle *model.FriendCircle) error
	Delete(ctx context.Context, id int64) error
	CountByOwner(ctx context.Context, ownerID int64) (int, error)

	AddMember(ctx context.Context, circleID, userID int64) error
	RemoveMember(ctx context.Context, circleID, userID int64) error
	IsMember(ctx context.Context, circleID, userID int64) (bool, error)
	ListMembers(ctx context.Context, circleID int64, page, pageSize int) ([]int64, error)
	CountMembers(ctx context.Context, circleID int64) (int, error)
}

type pgxCircleRepo struct {
	pool *pgxpool.Pool
}

func NewCircleRepository(pool *pgxpool.Pool) CircleRepository {
	return &pgxCircleRepo{pool: pool}
}

func (r *pgxCircleRepo) Create(ctx context.Context, circle *model.FriendCircle) error {
	query := `
		INSERT INTO friend_circles (name, description, owner_id)
		VALUES ($1, $2, $3)
		RETURNING id, member_count, created_at, updated_at
	`

	err := r.pool.QueryRow(ctx, query,
		circle.Name,
		circle.Description,
		circle.OwnerID,
	).Scan(&circle.ID, &circle.MemberCount, &circle.CreatedAt, &circle.UpdatedAt)

	if err != nil {
		return fmt.Errorf("create circle: %w", err)
	}

	// Add owner as first member
	_, err = r.pool.Exec(ctx,
		`INSERT INTO circle_members (circle_id, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		circle.ID, circle.OwnerID,
	)
	if err != nil {
		// Non-fatal: circle was created
		return nil
	}

	return nil
}

func (r *pgxCircleRepo) GetByID(ctx context.Context, id int64) (*model.FriendCircle, error) {
	query := `
		SELECT id, name, description, owner_id, member_count, created_at, updated_at
		FROM friend_circles
		WHERE id = $1 AND deleted_at IS NULL
	`

	circle := &model.FriendCircle{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&circle.ID,
		&circle.Name,
		&circle.Description,
		&circle.OwnerID,
		&circle.MemberCount,
		&circle.CreatedAt,
		&circle.UpdatedAt,
	)

	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, nil
		}
		return nil, fmt.Errorf("get circle: %w", err)
	}

	return circle, nil
}

func (r *pgxCircleRepo) ListByOwner(ctx context.Context, ownerID int64, page, pageSize int) ([]*model.FriendCircle, error) {
	offset := (page - 1) * pageSize

	query := `
		SELECT id, name, description, owner_id, member_count, created_at, updated_at
		FROM friend_circles
		WHERE owner_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.pool.Query(ctx, query, ownerID, pageSize, offset)
	if err != nil {
		return nil, fmt.Errorf("list circles by owner: %w", err)
	}
	defer rows.Close()

	return r.scanCircles(rows)
}

func (r *pgxCircleRepo) ListByMember(ctx context.Context, userID int64, page, pageSize int) ([]*model.FriendCircle, error) {
	offset := (page - 1) * pageSize

	query := `
		SELECT c.id, c.name, c.description, c.owner_id, c.member_count, c.created_at, c.updated_at
		FROM friend_circles c
		JOIN circle_members m ON c.id = m.circle_id
		WHERE m.user_id = $1 AND c.deleted_at IS NULL
		ORDER BY c.created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.pool.Query(ctx, query, userID, pageSize, offset)
	if err != nil {
		return nil, fmt.Errorf("list circles by member: %w", err)
	}
	defer rows.Close()

	return r.scanCircles(rows)
}

func (r *pgxCircleRepo) scanCircles(rows pgx.Rows) ([]*model.FriendCircle, error) {
	var circles []*model.FriendCircle
	for rows.Next() {
		circle := &model.FriendCircle{}
		err := rows.Scan(
			&circle.ID,
			&circle.Name,
			&circle.Description,
			&circle.OwnerID,
			&circle.MemberCount,
			&circle.CreatedAt,
			&circle.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan circle: %w", err)
		}
		circles = append(circles, circle)
	}

	return circles, rows.Err()
}

func (r *pgxCircleRepo) Update(ctx context.Context, circle *model.FriendCircle) error {
	query := `
		UPDATE friend_circles
		SET name = $1, description = $2, updated_at = NOW()
		WHERE id = $3 AND deleted_at IS NULL
		RETURNING updated_at
	`

	err := r.pool.QueryRow(ctx, query, circle.Name, circle.Description, circle.ID).Scan(&circle.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update circle: %w", err)
	}

	return nil
}

func (r *pgxCircleRepo) Delete(ctx context.Context, id int64) error {
	query := `UPDATE friend_circles SET deleted_at = NOW(), updated_at = NOW() WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete circle: %w", err)
	}
	return nil
}

func (r *pgxCircleRepo) CountByOwner(ctx context.Context, ownerID int64) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM friend_circles WHERE owner_id = $1 AND deleted_at IS NULL`,
		ownerID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count circles: %w", err)
	}
	return count, nil
}

func (r *pgxCircleRepo) AddMember(ctx context.Context, circleID, userID int64) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Insert member
	_, err = tx.Exec(ctx,
		`INSERT INTO circle_members (circle_id, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		circleID, userID,
	)
	if err != nil {
		return fmt.Errorf("add member: %w", err)
	}

	// Update count
	_, err = tx.Exec(ctx,
		`UPDATE friend_circles SET member_count = member_count + 1 WHERE id = $1`,
		circleID,
	)
	if err != nil {
		return fmt.Errorf("update member count: %w", err)
	}

	return tx.Commit(ctx)
}

func (r *pgxCircleRepo) RemoveMember(ctx context.Context, circleID, userID int64) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Delete member
	result, err := tx.Exec(ctx,
		`DELETE FROM circle_members WHERE circle_id = $1 AND user_id = $2`,
		circleID, userID,
	)
	if err != nil {
		return fmt.Errorf("remove member: %w", err)
	}

	if result.RowsAffected() == 0 {
		return nil // Not a member, nothing to do
	}

	// Update count
	_, err = tx.Exec(ctx,
		`UPDATE friend_circles SET member_count = GREATEST(member_count - 1, 0) WHERE id = $1`,
		circleID,
	)
	if err != nil {
		return fmt.Errorf("update member count: %w", err)
	}

	return tx.Commit(ctx)
}

func (r *pgxCircleRepo) IsMember(ctx context.Context, circleID, userID int64) (bool, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM circle_members WHERE circle_id = $1 AND user_id = $2`,
		circleID, userID,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check membership: %w", err)
	}
	return count > 0, nil
}

func (r *pgxCircleRepo) ListMembers(ctx context.Context, circleID int64, page, pageSize int) ([]int64, error) {
	offset := (page - 1) * pageSize

	rows, err := r.pool.Query(ctx,
		`SELECT user_id FROM circle_members WHERE circle_id = $1 ORDER BY joined_at DESC LIMIT $2 OFFSET $3`,
		circleID, pageSize, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("list members: %w", err)
	}
	defer rows.Close()

	var members []int64
	for rows.Next() {
		var userID int64
		if err := rows.Scan(&userID); err != nil {
			return nil, fmt.Errorf("scan member: %w", err)
		}
		members = append(members, userID)
	}

	return members, rows.Err()
}

func (r *pgxCircleRepo) CountMembers(ctx context.Context, circleID int64) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM circle_members WHERE circle_id = $1`,
		circleID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count members: %w", err)
	}
	return count, nil
}
