package server

import (
	"context"
	"encoding/json"

	"github.com/pigfall/tzzGoUtil/log"
	"github.com/pigfall/tzzGoUtil/net"
	"github.com/pigfall/yingying/proto"
	yy "github.com/pigfall/yingying"
	"github.com/pigfall/yingying/server/proto_handler"
)

type connCtrl struct {
	conns      map[string]yy.Transport
	tunIp      *net.IpWithMask
	ipPoolIfce ipPoolIfce
	rawLogger  log.Logger_Log
}

func newConnCtrl(ipPoolIfce ipPoolIfce, rawLogger log.Logger_Log) yy.ConnCtrl{
	return &connCtrl{
		conns:      make(map[string]yy.Transport),
		ipPoolIfce: ipPoolIfce,
		rawLogger:  rawLogger,
	}
}

func (this *connCtrl) GetConns()map[string]yy.Transport{
	return this.conns
}

func (this *connCtrl) Serve(
	ctx context.Context,
	// conn *ws.Conn,
	 conn yy.Transport,
	tunIfce net.TunIfce,
) error {
	logger := log.NewHelper("Serve", this.rawLogger, log.LevelDebug)
	clientVPNIpNet, err := this.ipPoolIfce.Take()
	if err != nil {
		logger.Error(err)
		return err
	}
	defer this.ipPoolIfce.Release(clientVPNIpNet)
	var clientVPNIpNetStr = clientVPNIpNet.String()
	this.conns[clientVPNIpNetStr] = conn
	defer delete(this.conns, clientVPNIpNetStr)

	err = connToTunIfce(ctx, this.rawLogger, conn, tunIfce, clientVPNIpNet)
	if err != nil {
		logger.Errorf("connToTunIfce return err %v", err)
		return err
	}
	return nil
}

func connToTunIfce(ctx context.Context, rawLogger log.Logger_Log, conn yy.Transport, tunIfce net.TunIfce, clientVPNIp *net.IpWithMask) error {
	logger := log.NewHelper("connToTunIfce", rawLogger, log.LevelDebug)
	for {
		msgType, msgBytes, err := conn.Read()
		if err != nil {
			logger.Error(err)
			return err
		}
		if msgType == yy.IpPacket { // proxy ip packet
			_, err = tunIfce.Write(msgBytes)
			if err != nil {
				logger.Error("write ip packet to tun ifce failed ", err)
				continue
			}
		} else {
			// handle custome proto
			msg := &proto.Msg{}
			err := json.Unmarshal(msgBytes, msg)
			if err != nil {
				logger.Error("parse custome msg failed %w", err)
				continue
			}
			res, body := proto_handler.NewHandler(clientVPNIp).Handle(
				ctx, rawLogger,
				msg,
			)
			if err := res.Err() ;err != nil{
				err = conn.WriteJSON(res)
				if err != nil {
					logger.Error(err)
				}
			}else{
				bodyByte,err := json.Marshal(body)
				if err != nil {
					logger.Error(err)
				} else {
					res.Body = bodyByte
					conn.WriteJSON(res)
				}
			}
				//
			
		}
	}
}
