package server

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func GetMetadataUniqueKey(ctx context.Context, key string) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Errorf(codes.DataLoss, "failed to get metadata")
	}
	if v, ok := md[key]; ok {
		if len(v) != 1 {
			return "", status.Errorf(codes.Unknown, "more than one value for metadata key")
		}
		return v[0], nil
	}

	return "", status.Error(codes.DataLoss, "missing key")
}
