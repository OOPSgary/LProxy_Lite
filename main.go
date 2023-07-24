package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"

	MinecraftNet "github.com/Tnze/go-mc/net"
	"github.com/Tnze/go-mc/net/packet"
)

type Config struct {
	Listen     int    `json:"Listen"`
	TargetIP   string `json:"TargetIP"`
	TargetPort int    `json:"TargetPort"`
	Rewrite    string `json:"RewritenAddress"`
	Motd       string `json:"Motd"`
	Favicon    string `json:"Favicon"`
}
type Motd struct {
	Version struct {
		Name     string `json:"name"`
		Protocol int    `json:"protocol"`
	} `json:"version"`
	Players struct {
		Max    int `json:"max"`
		Online int `json:"online"`
	} `json:"players"`
	Description struct {
		Text string `json:"text"`
	} `json:"description"`
	Favicon string `json:"favicon"`
}

var C Config

const Version = "1.0"

func main() {
	log.Println("[事件]Welcome to LProxy_Lite!")
	log.Println("Loading config from config.json...")
	Config_Load()
	Lis, err := net.ListenTCP("tcp4", &net.TCPAddr{
		IP:   nil,
		Port: int(C.Listen),
	})
	if err != nil {
		log.Panic("监听错误,ErrMSG:", err)
	}
	for {

		Conn, err := Lis.AcceptTCP()
		if err == nil {
			go Handler(Conn)
		} else {
			log.Println(err)
		}

	}
}
func Handler(Conn *net.TCPConn) {
	Conn_MC := MinecraftNet.WrapConn(Conn)
	defer Conn.Close()
	var p packet.Packet
	err := Conn_MC.ReadPacket(&p)
	if err != nil {
		log.Println("[错误]:", err)
		return
	}
	var (
		protocol  packet.VarInt
		hostname  packet.String
		port      packet.UnsignedShort
		nextState packet.Byte
	)
	err = p.Scan(&protocol, &hostname, &port, &nextState)
	if err != nil {
		log.Println("[错误]:", err)
		return
	}
	switch nextState {
	case 1:
		err = Conn_MC.ReadPacket(&p)
		if err != nil {
			log.Println("[错误]:", err)
			return
		}
		err = Conn_MC.WritePacket(motd(int(protocol)))
		if err != nil {
			log.Println("[错误]:", err)
			return
		}
		err = Conn_MC.ReadPacket(&p)
		if err != nil {
			log.Println("[错误]:", err)
			return
		}
		err = Conn_MC.WritePacket(p)
		if err != nil {
			log.Println("[错误]:", err)
			return
		}
	case 2:
		err = Conn_MC.ReadPacket(&p)
		if err != nil {
			log.Println("[错误]:", err)
			return
		}
		var Player packet.String
		err = p.Scan(&Player)
		if err != nil {
			log.Println("[错误]:", err)
			return
		}
		log.Printf("玩家:%s 正在登录\n", Player)
		remote, err := net.Dial("tcp", fmt.Sprintf("%v:%v", C.TargetIP, C.TargetPort))
		if err != nil {
			log.Println("[错误]:", err)
			return
		}
		defer remote.Close()
		remoteMC := MinecraftNet.WrapConn(remote)
		err = remoteMC.WritePacket(packet.Marshal(
			0x0,
			protocol,
			packet.String(func() string {
				if strings.HasSuffix(string(hostname), "\x00FML\x00") {
					return C.Rewrite + "\x00FML\x00"
				}
				return C.Rewrite + "\x00"
			}()),
			packet.UnsignedShort(C.TargetPort),
			packet.Byte(2),
		))
		if err != nil {
			log.Println("[错误]:", err)
			return
		}
		err = remoteMC.WritePacket(p)
		if err != nil {
			log.Println("[错误]:", err)
			return
		}
		go func() {
			defer Conn.Close()
			defer remote.Close()
			io.Copy(Conn, remote)
		}()

		io.Copy(remote, Conn)
	}
}
func Config_Load() {
	config_Json, err := os.ReadFile("Config.json")
	if err != nil {
		log.Println("[事件]配置没有找到……加载一个新的配置")
		Config_Gen()
		return
	}
	err = json.Unmarshal(config_Json, &C)
	if err != nil {
		log.Panic("[错误]解析配置错误……ErrMSG:", err)
	}
}
func Config_Gen() {
	file, err := os.Create("Config.json")
	if err != nil {
		log.Fatal("[错误] 创建Config:", err)
	}
	C = Config{
		Listen:     25565,
		TargetIP:   "mc.hypixel.net",
		TargetPort: 25565,
		Rewrite:    "mc.hypixel.net",
		Motd:       fmt.Sprintf("LPROXY_Lite V%v", Version),
		Favicon:    "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAEAAAABACAIAAAAlC+aJAAAAAXNSR0IArs4c6QAAAARnQU1BAACxjwv8YQUAAAAJcEhZcwAAEnQAABJ0Ad5mH3gAAAFCSURBVGhD7ZZRksIgEAVzED89xR5oj+hlPINn2CXQUUYSDYbUUOZ1jR+8gYQuwXK4nc6v669vJOBNhcCwDDM8aCzAeB2s2cZRBYimkMGXCRAV0JZAQgJzwwRRAW1fgSeYIYEPOKoAUQHtCFEB7f4F3sKabUjAGwl4cySBPpGANxLwRgLeSMAbCXjzXoB/P5ErqwIXojmYEiEafsKHqCl1AgGWZdih2WiKcmi0Y+URuvL+gvt3wngiZjOrYt6StXeA9xueT0VKl2BSayouMRux0IsQjYxud2jvw+cC2YV+EHKz9+E3fOjtQ4VAeaaJM2hY6O3DljuwtLOHJ8GerBJgO3Okp1gkUMNWgUB6UEZnAuxlxPzA5AMeBv0KjLDMksKJ/o5Qz0jAGwl4IwFvDiDQeUnAuyTgXRLwLgl4lwR863T+B2iQrVDfPXDxAAAAAElFTkSuQmCC",
	}
	nc, _ :=
		json.MarshalIndent(C, "", "    ")
	file.WriteString(string(nc))
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			return
		}
	}(file)
}
func motd(protocol int) packet.Packet {
	m := Motd{
		Version: struct {
			Name     string "json:\"name\""
			Protocol int    "json:\"protocol\""
		}{
			Name:     fmt.Sprintf("LProxyLite:%v", Version),
			Protocol: protocol,
		},
		Players: struct {
			Max    int "json:\"max\""
			Online int "json:\"online\""
		}{
			Max:    -1,
			Online: -1,
		},
		Description: struct {
			Text string "json:\"text\""
		}{
			Text: C.Motd,
		},
		Favicon: C.Favicon,
	}
	data, _ := json.Marshal(m)
	return packet.Marshal(
		0x0,
		packet.String(data),
	)
}
