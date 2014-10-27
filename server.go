package main

import (
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

//全局变量
// 对于关系：  IP  对于的 站点 手机号
var siteiptotel = make(map[string]string, 1000)

//记录站点  连续多少次 没有 返回请求数据 ， 达到10次 就断开链接
var sitetelnoreply = make(map[string]int, 1000)

//记录 每一个连接
var conns = make(map[string]net.Conn, 1000)

////////////////////////////////////////////////////////
//
//错误检查
//
////////////////////////////////////////////////////////
func checkError(err error, info string) (res bool) {
	datestr := time.Now().String()
	if err != nil {
		fmt.Println(info + "  " + err.Error() + "  " + datestr[0:16])
		return false
	}
	return true
}

////////////////////////////////////////////////////////
//
//服务器端接收数据线程
//参数：
//		数据连接 conn
////////////////////////////////////////////////////////
func Handler(conn net.Conn) {
	//获取访问者IP端口
	siteno := "0"
	ipstr := conn.RemoteAddr().String()
	fmt.Println("connected from", ipstr)
	buf := make([]byte, 1024)
	for {
		lenght, err := conn.Read(buf)
		if checkError(err, "Readerr: Close"+ipstr) == false {
			conn.Close()
			delete(conns, ipstr)
			delete(sitetelnoreply, siteiptotel[ipstr])
			delete(siteiptotel, ipstr)
			break
		}
		if lenght > 0 {
			buf[lenght] = 0
		}
		if siteno == "0" && lenght > 20 {
			siteno = string(buf[4:15])
			siteiptotel[ipstr] = siteno
		}
		//fmt.Println("Rec[", ipstr, "] Say :", string(buf[0:lenght]))
		reciveStr := string(buf[0:lenght])
		if lenght > 20 && crcok(buf) {
			//用于记录 站点有多少次 没有回复指令累计10次就断开链接
			sitetelnoreply[siteno] = 0

			setFiles(reciveStr, siteno)
			//这个站点累计0次没有回复
		}
	}
}

//定时群发 采集命令
func dssends() {
	//set := "030300000002C5E9"
	set := "010300320013A5C8"
	setBytes, _ := hex.DecodeString(set)
	fmt.Printf("% X", setBytes) // 03 03 00 00 00 32 C5 FD
	//每30秒发送一次命令
	tc := time.Tick(time.Second * 60) //返回一个time.C这个管道，1秒(time.Second)后会在此管道中放入一个时间点，

	for {
		<-tc
		for key, value := range conns {
			sitetelnoreply[siteiptotel[key]]++
			_, err := value.Write(setBytes)
			if err != nil {
				fmt.Println("write err", err.Error())
				value.Close()
				delete(conns, key)
				delete(sitetelnoreply, siteiptotel[key])
				delete(siteiptotel, key)
			} else if sitetelnoreply[siteiptotel[key]] > 10 {
				//fmt.Println(datestr[0:16]+" send:", key)
				//连续10次没有回馈就断开连接
				fmt.Println("noreply >10" + key)

				value.Close()
				delete(conns, key)
				delete(sitetelnoreply, siteiptotel[key])
				delete(siteiptotel, key)
			}
		}
		//fmt.Println(sitetelnoreply)

	}

}

////////////////////////////////////////////////////////
//
//启动服务器
//参数
//	端口 port
//
////////////////////////////////////////////////////////
func StartServer(port string) {
	service := ":" + port //strconv.Itoa(port);
	tcpAddr, err := net.ResolveTCPAddr("tcp4", service)
	checkError(err, "ResolveTCPAddr")
	l, err := net.ListenTCP("tcp", tcpAddr)
	checkError(err, "ListenTCP")

	//定时群发
	go dssends()
	for {
		datestr := time.Now().String()
		fmt.Println("Listening ..." + datestr[0:16] + ";")
		conn, err := l.Accept()
		checkError(err, "Accept")
		fmt.Println("Accepting ..." + datestr[0:16] + ";")
		conns[conn.RemoteAddr().String()] = conn
		siteiptotel[conn.RemoteAddr().String()] = ""
		//启动一个新线程
		go Handler(conn)

	}

}

////////////////////////////////////////////////////////
//
//主程序
//
////////////////////////////////////////////////////////
func main() {
	StartServer("5003")

}

/////////////////////////////////
//str 写入文件的内容
// filename 写入的文件名称
//
//字符串写入文件
func setFiles(str string, wjname string) {
	datestr := time.Now().String()
	//替换 ： 为 .
	wjname = datestr[5:7] + datestr[8:10] + "." + strings.Replace(wjname, ":", ".", -1) + ".log"
	f, err := os.OpenFile("/home/Data/dtu/"+"/"+wjname, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		checkError(err, "OpenFile")
	}
	defer f.Close()
	setstr := datestr[0:16] + ":" + str + "\r\n"
	f.Write([]byte(setstr))
	//备份用的
	fb, err := os.OpenFile("/home/Data/dtu/"+wjname+".bak", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		checkError(err, "OpenFile")
	}
	defer fb.Close()
	fb.Write([]byte(setstr))

}

//返回CRC校验是否成功
func crcok(data []byte) bool {
	packet_crc := crc(data[:len(data)-2])
	return string(packet_crc) == string(data[len(data)-2:])
}

// 传入 modbus 数据内容，返回 crc校验码
func crc(data []byte) []byte {
	var crc16 uint16 = 0xffff
	l := len(data)
	for i := 0; i < l; i++ {
		crc16 ^= uint16(data[i])
		for j := 0; j < 8; j++ {
			if crc16&0x0001 > 0 {
				crc16 = (crc16 >> 1) ^ 0xA001
			} else {
				crc16 >>= 1
			}
		}
	}
	packet := make([]byte, 2)
	packet[0] = byte(crc16 & 0xff)
	packet[1] = byte(crc16 >> 8)

	return packet
}
