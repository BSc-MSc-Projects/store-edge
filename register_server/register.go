/* This file contains the logic for the register node. */
package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"

	pb_register "github.com/piercirocaliandro/sdcc-edge-computing/pb_register"
	"google.golang.org/grpc"
)

type Socket struct {
	ip_addr string
	port    int
}

type registerServer struct {
	pb_register.UnimplementedRegisterServer          // interface for the grpc server side
	peers                                   []Socket // list of peers connected
	clients                                 []Socket // list of clients connected
}

var ttl map[string]float32 // to check if the StorageServer is still alive
var peer_index int         // number of peers connected to a certain StorageServer

/*--------------------------------------------- RPC functions -----------------------------------------------------------------*/

func (s *registerServer) GetCloseStorageServers(ctx context.Context, cmsg *pb_register.ClientMsg) (*pb_register.ListOfStorageServerMsg, error) {
	fmt.Println("A new client connected")
	addClientInList(cmsg, s)
	return getStorageServers(cmsg, s), nil
}

func (s *registerServer) GetListOfPeers(ctx context.Context, smsg *pb_register.StorageServerMsg) (*pb_register.ListOfStorageServerMsg, error) {
	fmt.Println("A new StorageServer connected")
	addServerInList(smsg, s)
	return getListOfStorageServer(smsg, s), nil
}

func (s *registerServer) Heartbeat(ctx context.Context, smsg *pb_register.StorageServerMsg) (*pb_register.ListOfStorageServerMsg, error) {
	fmt.Println("Heartbeat received from", smsg.IpAddr, smsg.Port)
	ttl[smsg.IpAddr+fmt.Sprint(smsg.Port)] = 90.0
	return getListOfStorageServer(smsg, s), nil
}

/*--------------------------------------------- Utility functions -----------------------------------------------------------*/

func addClientInList(client *pb_register.ClientMsg, s *registerServer) {
	socket := Socket{
		ip_addr: client.IpAddr,
		port:    int(client.Port),
	}
	s.clients = append(s.clients, socket)
}

func addServerInList(server *pb_register.StorageServerMsg, s *registerServer) {
	socket := Socket{
		ip_addr: server.IpAddr,
		port:    int(server.Port),
	}
	s.peers = append(s.peers, socket)
	ttl[server.IpAddr+fmt.Sprint(server.Port)] = 90.0
}

func getStorageServers(client *pb_register.ClientMsg, s *registerServer) *pb_register.ListOfStorageServerMsg {
	var ipAddrList []string
	var portList []int32
	for i := 0; i < len(s.peers); i++ {
		socket := s.peers[(peer_index+i)%(len(s.peers))]
		ipAddrList = append(ipAddrList, socket.ip_addr)
		portList = append(portList, int32(socket.port))
	}
	serverSocket := pb_register.ListOfStorageServerMsg{
		IpAddr: ipAddrList,
		Port:   portList,
	}
	peer_index += 1
	return &serverSocket
}

func getListOfStorageServer(server *pb_register.StorageServerMsg, s *registerServer) *pb_register.ListOfStorageServerMsg {
	var ipAddrList []string
	var portList []int32

	for i := 0; i < len(s.peers); i++ {
		socket := s.peers[(peer_index+i)%(len(s.peers))]
		if socket.ip_addr != server.IpAddr || socket.port != int(server.Port) {
			ipAddrList = append(ipAddrList, socket.ip_addr)
			portList = append(portList, int32(socket.port))
		}
	}
	serverSocket := pb_register.ListOfStorageServerMsg{
		IpAddr: ipAddrList,
		Port:   portList,
	}
	peer_index += 1
	return &serverSocket
}

// Constructor for the rpc server
func newServer() *registerServer {
	s := &registerServer{}
	return s
}

func getMyIp() string {
	var ret string

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		log.Fatalf("Oops: " + err.Error() + "\n")
		os.Exit(1)
	}

	for _, ip := range addrs {
		if ipnet, ok := ip.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if len(ip.String()) <= 20 {
				ipAddr := ip.String()
				ret = strings.Split(ipAddr, "/")[0]
			}
		}
	}
	return ret
}

func decrementTTL(s *registerServer) {
	for {
		time.Sleep(30000000000.0)
		for k := range ttl {
			ttl[k] = ttl[k] - 30.0
			if ttl[k] <= 0.0 {
				for index, elem := range s.peers {
					if (elem.ip_addr + fmt.Sprint(elem.port)) == k {
						s.peers = append(s.peers[:index], s.peers[index+1:]...)
					}
				}
				delete(ttl, k)
			}
		}
	}
}

// Main fucntion
func main() {
	port := 1234
	ipAddr := getMyIp()
	fmt.Print("Listen on Ip:" + ipAddr + "\n")
	lis, err := net.Listen("tcp", fmt.Sprintf(ipAddr+":%d", port))
	if err != nil {
		log.Fatalf("Error during listening: %v", err)
	}
	ttl = make(map[string]float32)
	peer_index = 0

	// start the register server to receive upcoming requests from peers
	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	s := newServer()
	pb_register.RegisterRegisterServer(grpcServer, s)
	fmt.Println("The register server is listening on port:", port)

	go decrementTTL(s)
	grpcServer.Serve(lis)
}
