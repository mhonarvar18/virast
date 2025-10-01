package followerapp

import (
	"context"
	"errors"
	"virast/internal/config"
	followerEntity "virast/internal/core/follower"
	followerPort "virast/internal/ports/follower"

	"github.com/gofrs/uuid"
	"go.uber.org/zap"
)

type FollowerService struct {
	FollowerRepository followerPort.FollowerRepository
}

func NewFollowerService(repo followerPort.FollowerRepository) *FollowerService {
	return &FollowerService{
		FollowerRepository: repo,
	}
}

func (s *FollowerService) FollowUser(ctx context.Context, followerID, followeeID string) error {
	if followerID == followeeID {
		config.Logger.Warn("⚠️ Cannot follow yourself", zap.String("userID", followerID))
		return errors.New("cannot follow yourself")
	}

	f := &followerEntity.Follower{
		ID:         uuid.Must(uuid.NewV4()),
		UserID:     uuid.FromStringOrNil(followeeID),
		FollowerID: uuid.FromStringOrNil(followerID),
	}

	_, err := s.FollowerRepository.FollowUser(ctx, f)
	return err
}

func (s *FollowerService) UnfollowUser(ctx context.Context, followerID, followeeID string) error {
	return s.FollowerRepository.UnfollowUser(ctx, followerID, followeeID)
}

func (s *FollowerService) GetFollowersByUserID(ctx context.Context, userID string) ([]*followerPort.FollowerDTO, error) {
	followers, err := s.FollowerRepository.GetFollowersByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	var followerDTOs []*followerPort.FollowerDTO
	for _, f := range followers {
		followerDTOs = append(followerDTOs, &followerPort.FollowerDTO{
			ID:         f.ID.String(),
			UserID:     f.UserID.String(),
			FollowerID: f.FollowerID.String(),
		})
	}

	// اگر slice خالی یا nil بود، مقداردهی به یک آرایه خالی
	if followerDTOs == nil {
		followerDTOs = []*followerPort.FollowerDTO{}
	}

	return followerDTOs, nil
}

func (s *FollowerService) GetFollowingByUserID(ctx context.Context, userID string) ([]*followerPort.FollowerDTO, error) {
	following, err := s.FollowerRepository.GetFollowingByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	var followingDTOs []*followerPort.FollowerDTO
	for _, f := range following {
		followingDTOs = append(followingDTOs, &followerPort.FollowerDTO{
			ID:         f.ID.String(),
			UserID:     f.UserID.String(),
			FollowerID: f.FollowerID.String(),
		})
	}

	// اگر slice خالی یا nil بود، مقداردهی به یک آرایه خالی
	if followingDTOs == nil {
		followingDTOs = []*followerPort.FollowerDTO{}
	}

	return followingDTOs, nil
}

func (s *FollowerService) IsFollowing(ctx context.Context, followerID, followeeID string) (bool, error) {
	following, err := s.FollowerRepository.GetFollowingByUserID(ctx, followerID)
	if err != nil {
		return false, err
	}
	for _, f := range following {
		if f.UserID == uuid.FromStringOrNil(followeeID) {
			return true, nil
		}
	}
	return false, nil
}
