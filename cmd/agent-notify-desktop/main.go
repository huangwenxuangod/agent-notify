package main

import (
	"context"
	"embed"
	"os"
	"github.com/hellolib/agent-notify/internal/bridge"
	"github.com/hellolib/agent-notify/internal/common"
	"github.com/hellolib/agent-notify/internal/config"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
)

//go:embed all:frontend
var assets embed.FS

type App struct{ service *bridge.Service }
func NewApp() (*App,error){cp,e:=config.DefaultPath();if e!=nil{return nil,e};sp,e:=config.StatePath();if e!=nil{return nil,e};lp,e:=config.LogPath();if e!=nil{return nil,e};svc,e:=bridge.NewService(bridge.Options{ConfigPath:cp,StatePath:sp,LogPath:lp,BinaryPath:common.ResolveBinaryPath("")});if e!=nil{return nil,e};return &App{service:svc},nil}
func(a *App)Startup(ctx context.Context){ }
func(a *App)Scan()([]bridge.AgentStatus,error){return a.service.ScanAgents()}
func(a *App)Install(agents []string,scope string)(bridge.SetupResult,error){return a.service.InstallAgents(bridge.SetupRequest{Agents:agents,Scope:scope})}
func(a *App)Uninstall(agents []string,scope string)(bridge.SetupResult,error){return a.service.UninstallAgents(bridge.SetupRequest{Agents:agents,Scope:scope})}
func(a *App)Events()([]interface{},error){events,err:=a.service.ListEvents(100);if err!=nil{return nil,err};out:=make([]interface{},len(events));for i,v:=range events{out[i]=v};return out,nil}
func main(){app,err:=NewApp();if err!=nil{panic(err)};if err:=wails.Run(&options.App{Title:"Agent Notify",Width:980,Height:700,AssetServer:options.AssetServer{Assets:assets},Bind:[]interface{}{app}});err!=nil{os.Exit(1)}}
