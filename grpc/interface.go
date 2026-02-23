package grpc

import (
	"context"
	pb "paymentfc/pb/proto"
)

type UserClientInterface interface {
	GetUserInfoByUserId(ctx context.Context, userId int64) (*pb.GetUserInfoByUserIdResponse, error)
	ValidateToken(ctx context.Context, token string) (*pb.ValidateTokenResponse, error)
	Close() error
}
