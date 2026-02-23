package grpc

import (
	"context"
	"paymentfc/log"
	pb "paymentfc/pb/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type UserClient struct {
	client pb.UserServiceClient
	conn   *grpc.ClientConn
}

func NewUserClient(address string) (*UserClient, error) {
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Logger.Error().Err(err).Str("address", address).Msg("Failed to connect to user service")
		return nil, err
	}

	client := pb.NewUserServiceClient(conn)
	log.Logger.Info().Str("address", address).Msg("Connected to user gRPC service")

	return &UserClient{
		client: client,
		conn:   conn,
	}, nil
}

func (c *UserClient) Close() error {
	return c.conn.Close()
}

func (c *UserClient) GetUserInfoByUserId(ctx context.Context, userId int64) (*pb.GetUserInfoByUserIdResponse, error) {
	resp, err := c.client.GetUserInfoByUserId(ctx, &pb.GetUserInfoByUserIdRequest{
		UserId: userId,
	})
	if err != nil {
		log.Logger.Error().Err(err).Int64("user_id", userId).Msg("Failed to get user info")
		return nil, err
	}
	return resp, nil
}

func (c *UserClient) ValidateToken(ctx context.Context, token string) (*pb.ValidateTokenResponse, error) {
	resp, err := c.client.ValidateToken(ctx, &pb.ValidateTokenRequest{
		Token: token,
	})
	if err != nil {
		log.Logger.Error().Err(err).Msg("Failed to validate token")
		return nil, err
	}
	return resp, nil
}
