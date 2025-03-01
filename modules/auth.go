package modules

import (
	"context"
	"evolve/config"
	pb "evolve/proto"
	"evolve/util"
	"fmt"
	"net/http"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func Auth(req *http.Request) (map[string]string, error) {
	var logger = util.NewLogger()
	logger.Info("Auth called.")

	token, err := req.Cookie("t")
	if err != nil {
		logger.Error(fmt.Sprintf("No token found in request: %v", err))
		return nil, fmt.Errorf("unauthorized")
	}

	// TODO: Verify the security level of gRPC connection.

	// Verify the token via a gRPC call
	// to the auth micro-service.
	authConn, err := grpc.NewClient(config.AUTH_GRPC_ADDRESS, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to create gRPC client: %v", err))
		return nil, fmt.Errorf("something went wrong")
	}
	defer authConn.Close()

	authClient := pb.NewAuthenticateClient(authConn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	r, err := authClient.Auth(ctx, &pb.TokenValidateRequest{Token: token.Value})
	if err != nil {
		logger.Error(fmt.Sprintf("gRPC call failed: %v", err))
		return nil, fmt.Errorf("something went wrong")
	}

	if !r.GetValid() {
		return nil, fmt.Errorf("unauthorized")
	}

	return map[string]string{
		"id":       r.GetId(),
		"userName": r.GetUserName(),
		"fullName": r.GetFullName(),
		"email":    r.GetEmail(),
		"role":     r.GetRole(),
	}, nil

}
