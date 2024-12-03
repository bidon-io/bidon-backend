package grpcserver

import (
	"context"
	pb "github.com/bidon-io/bidon-backend/pkg/proto/bidon/v1"
	v3 "github.com/bidon-io/bidon-backend/pkg/proto/com/iabtechlab/openrtb/v3"
)

type Server struct {
	pb.UnimplementedBiddingServiceServer
}

func (s *Server) Bid(context.Context, *v3.Openrtb) (*v3.Openrtb, error) {
	return &v3.Openrtb{}, nil
}
