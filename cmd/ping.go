package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/h2ik/go-grpc-pong/pb"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	pongAddr string
	interval time.Duration
)

var PingCmd = &cobra.Command{
	Use:   "ping",
	Short: "Start the ping daemon",
	Long:  "Start the ping daemon, which repeatedly sends Ping RPCs to the pong server.",
	RunE:  runPing,
}

func init() {
	PingCmd.Flags().StringVar(&pongAddr, "addr", "localhost:50051", "Address of the pong server (host:port)")
	PingCmd.Flags().DurationVar(&interval, "interval", 1*time.Second, "Interval between pings")
}

func runPing(cmd *cobra.Command, args []string) error {
	log.Printf("Connecting to pong server at %s", pongAddr)

	conn, err := grpc.NewClient(pongAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer conn.Close()

	client := pb.NewPongServiceClient(conn)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Printf("Ping daemon started — sending pings every %s", interval)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Ping daemon stopped")
			return nil
		case t := <-ticker.C:
			req := &pb.PingRequest{
				Message:   "ping",
				Timestamp: t.UnixNano(),
			}
			resp, err := client.Ping(ctx, req)
			if err != nil {
				log.Printf("Ping error: %v", err)
				continue
			}
			rtt := time.Since(time.Unix(0, req.Timestamp))
			log.Printf("Pong received: %q  rtt=%s", resp.Message, rtt.Round(time.Microsecond))
		}
	}
}
