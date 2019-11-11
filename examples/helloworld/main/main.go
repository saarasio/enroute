package main

import (
	"flag"
	"fmt"
	"context"
	"log"
	"time"
	"net"
    "os"

	"google.golang.org/grpc"
	"github.com/saarasio/enroute/examples/helloworld/pb"
)

type server struct {
	port string
}

func (s *server) HelloWorld(ctx context.Context, in *pb.Ping) (*pb.Ping, error) {
	log.Printf("got: [%s]", in.GetPing())
	return in, nil
}

// Example 
// ./main -role server -port 50055
// ./main -role client -host localhost -port 50055 -id 2
// ./main -role client -host localhost -port 50055 -id 1

func main() {

    var host, port, role, id string

    flag.StringVar(&host, "host", "localhost", "host to run - dial host for client")
    flag.StringVar(&port, "port", "50001", "port to run - listen port for server/dial port for client")
    flag.StringVar(&role, "role", "", "run with role - client/server")
    flag.StringVar(&id, "id", "1", "run with client-id")
    flag.Parse()

    switch role {
    case "client":
        server_url := host + ":" + port
        conn, err := grpc.Dial(
            server_url, grpc.WithInsecure(),
        )
        if err != nil {
            log.Panicf("dial err: %s", err)
        }
        defer conn.Close()

        client := pb.NewEchoClient(conn)
        for {
            message := fmt.Sprintf("%v:%v", id, time.Now().Second())
            got, err := client.HelloWorld(context.Background(), &pb.Ping{Ping: message})
            if err != nil {
                log.Printf("error: %s", err)
                time.Sleep(time.Second * 5)
                continue
            }
            log.Printf("send: %s", got.GetPing())
            time.Sleep(time.Second)
        }

    case "server":
        log.Printf("Running server on : %s", port)
        lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
        if err != nil {
            log.Fatalf("failed to listen: %v", err)
        }
        s := grpc.NewServer()
        pb.RegisterEchoServer(s, &server{port})
        if err := s.Serve(lis); err != nil {
            log.Fatalf("failed to serve: %v", err)
        }

    default:
        fmt.Printf("Specify role using - '-role client' or '-role server'\n")
        os.Exit(1)
    }
}
