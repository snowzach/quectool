package prober

import (
	"context"
	"fmt"
	"time"

	probing "github.com/prometheus-community/pro-bing"
	"github.com/snowzach/quectool/quectool"
)

type Response struct {
	Success    bool              `json:"success"`
	StatusCode int               `json:"status_code,omitempty"`
	Duration   quectool.Duration `json:"duration,omitempty"`
}

func ProbePing(ctx context.Context, target string, timeout time.Duration) (Response, error) {

	c := make(chan *probing.Packet)

	pinger, err := probing.NewPinger(target)
	if err != nil {
		return Response{}, fmt.Errorf("unable to create pinger: %v", err)
	}
	pinger.Interval = time.Second
	pinger.Count = 5
	pinger.OnRecv = func(pkt *probing.Packet) {
		select {
		case c <- pkt:
		default:
		}

	}
	go pinger.Run()
	defer pinger.Stop()

	select {
	case <-ctx.Done():
		return Response{}, ctx.Err()
	case pkt := <-c:
		return Response{
			Success:  true,
			Duration: quectool.Duration(pkt.Rtt),
		}, nil
	case <-time.After(timeout):
		return Response{
			Success: false,
		}, nil
	}

}

func ProbeHTTP(ctx context.Context, target string, timeout time.Duration) (Response, error) {

	type callerResponse struct {
		StatusCode      int
		IsValidResponse bool
		Duration        time.Duration
	}

	c := make(chan callerResponse)

	caller := probing.NewHttpCaller(target,
		probing.WithHTTPCallerCallFrequency(time.Second),
		probing.WithHTTPCallerOnResp(func(suite *probing.TraceSuite, info *probing.HTTPCallInfo) {
			c <- callerResponse{
				StatusCode:      info.StatusCode,
				IsValidResponse: info.IsValidResponse,
				Duration:        suite.GetGeneralEnd().Sub(suite.GetGeneralStart()),
			}
		}),
	)
	go caller.Run()
	defer caller.Stop()

	select {
	case <-ctx.Done():
		return Response{}, ctx.Err()
	case result := <-c:
		return Response{
			Success:    result.IsValidResponse,
			StatusCode: result.StatusCode,
			Duration:   quectool.Duration(result.Duration),
		}, nil
	case <-time.After(timeout):
		return Response{
			Success: false,
		}, nil
	}
}
