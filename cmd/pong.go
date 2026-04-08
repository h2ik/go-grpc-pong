package cmd

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/h2ik/go-grpc-pong/pb"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var listenAddr string

var PongCmd = &cobra.Command{
	Use:   "pong",
	Short: "Start the pong daemon",
	Long:  "Start the pong daemon, which listens for Ping RPCs and replies with Pong responses.",
	RunE:  runPong,
}

func init() {
	PongCmd.Flags().StringVar(&listenAddr, "addr", ":50051", "Address to listen on (host:port)")
}

// pongServer implements pb.PongServiceServer.
type pongServer struct {
	pb.UnimplementedPongServiceServer
}

func (s *pongServer) Ping(_ context.Context, req *pb.PingRequest) (*pb.PongResponse, error) {
	log.Printf("Received ping: %q", req.Message)
	return &pb.PongResponse{
		Message:   "pong",
		Timestamp: time.Now().UnixNano(),
	}, nil
}

func runPong(cmd *cobra.Command, args []string) error {
	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", listenAddr, err)
	}

	srv := grpc.NewServer()
	pb.RegisterPongServiceServer(srv, &pongServer{})
	reflection.Register(srv)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()
		log.Println("Shutting down pong server…")
		srv.GracefulStop()
	}()

	log.Printf("Pong daemon listening on %s", lis.Addr())
	if err := srv.Serve(lis); err != nil {
		return fmt.Errorf("server error: %w", err)
	}
	return nil
}
