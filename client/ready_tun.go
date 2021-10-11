package client

import(
	"context"
	"fmt"
	stdnet "net"
	"github.com/pigfall/tzzGoUtil/net"
	"github.com/pigfall/tzzGoUtil/async"
	"github.com/pigfall/tzzGoUtil/log"
  yy "github.com/pigfall/yingying"
	water_wrap "github.com/pigfall/tzzGoUtil/net/water_tun_wrap"
)


func readyTun(
	ctx context.Context,
	logger log.LoggerLite,
	tunIp *net.IpWithMask,
	asyncCtrl *async.Ctrl,
	tp yy.Transport,
	serverIp stdnet.IP,
)(tunIfce net.TunIfce,err error){
	tun,err := water_wrap.NewTun()
	if err != nil{
		err = fmt.Errorf("Create tun ifce failed %v",err)
		logger.Error(err)
		return nil,err
	}
	err = tun.SetIp(tunIp.String())
	if err != nil{
		logger.Error("Set tun ifce set failed %w",err)
		return nil,err
	}
	tunIfce = tun
	asyncCtrl.AppendCancelFuncs(func(){tun.Close()})
	asyncCtrl.AppendCancelFuncs(func(){
		net.DelRoute(serverIp)
	})
	asyncCtrl.AsyncDo(
		ctx,
		func(ctx context.Context){
			var buf = make([]byte,1024*4)
			for {
				n,err := tun.Read(buf)
				if err != nil{
					logger.Error(err)
					return
				}
				err = tp.WriteIpPacket(buf[:n])
				if err != nil{
					logger.Error(err)
				}
			}
		},
	)
	logger.Info("Create tun success")
	logger.Info("Setting route table")
	defaultRoute,err := net.GetDefaultRouteRule()
	if err != nil{
		err = fmt.Errorf("Get default route from kernel failed: %w",err)
		logger.Error(err)
		return nil,err
	}
	net.DelRoute(serverIp)
	err = net.AddRoute(serverIp,defaultRoute.DevName,defaultRoute.Via)
	if err != nil{
		logger.Error(err)
		return nil,err
	}
	targetA,err := net.FromIpSlashMask("0.0.0.0/1")
	if err != nil{
		return nil,err
	}
	targetB,err := net.FromIpSlashMask("128.0.0.0/1")
	if err != nil{
		return nil,err
	}
	err = net.AddRouteIpNet(targetA,tunIfce.Name(),nil)
	if err != nil{
		logger.Error("Set route table failed %v",err)
		return nil,err
	}
	err = net.AddRouteIpNet(targetB,tunIfce.Name(),nil)
	if err != nil{
		logger.Error("Set route table failed %v",err)
		return nil,err
	}
	logger.Info("Setted route table")
	return tunIfce,nil
}
