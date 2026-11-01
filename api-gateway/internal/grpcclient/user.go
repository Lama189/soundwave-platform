package grpcclient

import (
	"context"
	"fmt"
	"time"

	userpb "github.com/Lama189/soundwave-platform/gen/go/user/v1"
	"github.com/google/uuid"
	"google.golang.org/grpc"
)

type UserClient struct {
	grpcClient userpb.UserServiceClient
	timeout    time.Duration
}

func NewUserClient(conn grpc.ClientConnInterface, defaultTimeout time.Duration) *UserClient {
	if defaultTimeout <= 0 {
		defaultTimeout = 5 * time.Second
	}

	return &UserClient{
		grpcClient: userpb.NewUserServiceClient(conn),
		timeout:    defaultTimeout,
	}
}

func (c *UserClient) Register(ctx context.Context, req RegisterRequest) (*UserResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	pbReq := &userpb.RegisterRequest{
		Phone:    req.Phone,
		Password: req.Password,
	}

	pbRes, err := c.grpcClient.Register(ctx, pbReq)
	if err != nil {
		return nil, err
	}

	userID, err := uuid.Parse(pbRes.GetUser().GetId())
	if err != nil {
		return nil, fmt.Errorf("invalid user id returned from gRPC: %w", err)
	}

	return &UserResponse{
		ID:        userID,
		Phone:     pbRes.GetUser().GetPhone(),
		Role:      pbRes.GetUser().GetRole(),
		CreatedAt: pbRes.GetUser().CreatedAt.AsTime(),
	}, nil
}

func (c *UserClient) Login(ctx context.Context, req LoginRequest) (*AuthResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	pbReq := &userpb.LoginRequest{
		Phone:    req.Phone,
		Password: req.Password,
	}

	pbRes, err := c.grpcClient.Login(ctx, pbReq)
	if err != nil {
		return nil, err
	}

	userID, err := uuid.Parse(pbRes.GetUser().GetId())
	if err != nil {
		return nil, fmt.Errorf("invalid user id returned from gRPC: %w", err)
	}

	return &AuthResponse{
		User: UserResponse{
			ID:        userID,
			Phone:     pbRes.GetUser().GetPhone(),
			Role:      pbRes.GetUser().GetRole(),
			CreatedAt: pbRes.GetUser().CreatedAt.AsTime(),
		},
		AccessToken:  pbRes.GetAccessToken(),
		RefreshToken: pbRes.GetRefreshToken(),
	}, nil
}

func (c *UserClient) Refresh(ctx context.Context, req RefreshRequest) (*RefreshResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	pbReq := &userpb.RefreshRequest{
		RefreshToken: req.RefreshToken,
	}

	pbRes, err := c.grpcClient.Refresh(ctx, pbReq)
	if err != nil {
		return nil, err
	}

	return &RefreshResponse{
		AccessToken:  pbRes.GetAccessToken(),
		RefreshToken: pbRes.GetRefreshToken(),
	}, nil
}

func (c *UserClient) GetMe(ctx context.Context, userID uuid.UUID) (*UserResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	pbReq := &userpb.GetUserRequest{
		Id: userID.String(),
	}

	pbRes, err := c.grpcClient.GetUser(ctx, pbReq)
	if err != nil {
		return nil, err
	}

	parsedID, err := uuid.Parse(pbRes.GetUser().GetId())
	if err != nil {
		return nil, fmt.Errorf("invalid user id returned from gRPC: %w", err)
	}

	return &UserResponse{
		ID:        parsedID,
		Phone:     pbRes.GetUser().GetPhone(),
		Role:      pbRes.GetUser().GetRole(),
		CreatedAt: pbRes.GetUser().GetCreatedAt().AsTime(),
	}, nil
}

func (c *UserClient) UpdateUserRole(ctx context.Context, userID uuid.UUID, role string) (*UserResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	pbReq := &userpb.UpdateUserRoleRequest{
		UserId: userID.String(),
		Role:   role,
	}

	pbRes, err := c.grpcClient.UpdateUserRole(ctx, pbReq)
	if err != nil {
		return nil, err
	}

	parsedID, err := uuid.Parse(pbRes.GetUser().GetId())
	if err != nil {
		return nil, fmt.Errorf("invalid user id returned from gRPC: %w", err)
	}

	return &UserResponse{
		ID:        parsedID,
		Phone:     pbRes.GetUser().GetPhone(),
		Role:      pbRes.GetUser().GetRole(),
		CreatedAt: pbRes.GetUser().GetCreatedAt().AsTime(),
	}, nil
}
