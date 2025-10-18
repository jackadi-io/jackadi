package server

import (
	"context"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestGetMetadataUniqueKey(t *testing.T) {
	tests := []struct {
		name     string
		ctx      context.Context
		key      string
		want     string
		wantCode codes.Code
	}{
		{
			name: "valid single value",
			ctx: metadata.NewIncomingContext(
				context.Background(),
				metadata.Pairs("test-key", "test-value"),
			),
			key:  "test-key",
			want: "test-value",
		},
		{
			name: "multiple keys single value each",
			ctx: metadata.NewIncomingContext(
				context.Background(),
				metadata.Pairs("key1", "value1", "key2", "value2"),
			),
			key:  "key2",
			want: "value2",
		},
		{
			name:     "no metadata in context",
			ctx:      context.Background(),
			key:      "test-key",
			wantCode: codes.DataLoss,
		},
		{
			name: "missing key",
			ctx: metadata.NewIncomingContext(
				context.Background(),
				metadata.Pairs("other-key", "value"),
			),
			key:      "test-key",
			wantCode: codes.DataLoss,
		},
		{
			name: "multiple values for key",
			ctx: metadata.NewIncomingContext(
				context.Background(),
				metadata.MD{"test-key": []string{"value1", "value2"}},
			),
			key:      "test-key",
			wantCode: codes.Unknown,
		},
		{
			name: "empty value",
			ctx: metadata.NewIncomingContext(
				context.Background(),
				metadata.Pairs("test-key", ""),
			),
			key:  "test-key",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetMetadataUniqueKey(tt.ctx, tt.key)

			if tt.wantCode != 0 {
				if err == nil {
					t.Errorf("GetMetadataUniqueKey() expected error, got nil")
					return
				}
				st, ok := status.FromError(err)
				if !ok {
					t.Errorf("GetMetadataUniqueKey() error is not a status error: %v", err)
					return
				}
				if st.Code() != tt.wantCode {
					t.Errorf("GetMetadataUniqueKey() error code = %v, want %v", st.Code(), tt.wantCode)
				}
				return
			}

			if err != nil {
				t.Errorf("GetMetadataUniqueKey() unexpected error: %v", err)
				return
			}

			if got != tt.want {
				t.Errorf("GetMetadataUniqueKey() = %v, want %v", got, tt.want)
			}
		})
	}
}
