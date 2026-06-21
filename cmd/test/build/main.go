package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/EmptyDea-Team/bedrock-world-operator/chunk"
	"github.com/Yeah114/Fatalder/define"
	"github.com/Yeah114/Fatalder/frame"
	"github.com/Yeah114/Fatalder/frame/task/build"
)

var (
	sourceWorldPath = "VOH3-0主城@[0,0,0]~[170,320,220].mcworld"

	sourceDimension = define.Dimension(define.DimensionIDOverworld)
	targetDimension = define.Dimension(define.DimensionIDOverworld)

	speed          = 8000
	chunkGroupSide = 2
)

var (
	sourceStartPos = define.BlockPos{0, 0, 0}
	sourceEndPos   = define.BlockPos{170, 320, 220}
	targetStartPos = define.BlockPos{1200, 0, 1200}
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx := context.Background()

	frame := frame.FrameConfig{
		ClientConfig: frame.ClientConfig{
			AuthServer:     "http://127.0.0.1:8080",
			UserToken:      `{"emulator":1,"is_guest":false,"mac_addr":"0ffbc255e3015d3880142bfd6fdefad4","ram":"1035337728","rom":"134208294912","sauth_json":"{\"aim_info\":\"{\\\"aim\\\":\\\"127.0.0.1\\\",\\\"country\\\":\\\"CN\\\",\\\"tz\\\":\\\"+0800\\\",\\\"tzid\\\":\\\"Asia/Shanghai\\\",\\\"celluar_ip\\\":\\\"\\\",\\\"operator\\\":\\\"\\\",\\\"is_vpn_enabled\\\":false}\",\"app_channel\":\"4399com\",\"client_login_sn\":\"57f9ee2359184384915e0e8a05885881\",\"deviceid\":\"57f9ee2359184384915e0e8a05885881\",\"gameid\":\"x19\",\"gas_token\":\"\",\"get_access_token\":\"1\",\"ip\":\"127.0.0.1\",\"is_unisdk_guest\":0,\"login_channel\":\"4399com\",\"platform\":\"ad\",\"realname\":\"{\\\"realname_type\\\":\\\"0\\\"}\",\"sdk_version\":\"1.0.0\",\"sdkuid\":\"1361225243\",\"sessionid\":\"1361225243|1ff2124b4da965fc21cea41bf2ccab3e|44770||e12ec03e0edb30304aec7dab8a465e56|c10b25a84904f5361b1e9356d669161b|1783335716|4399\",\"source_app_channel\":\"4399com\",\"source_platform\":\"ad\",\"udid\":\"7ac1f87f59205290\"}"}`,
			ServerCode:     "48285363",
			ServerPassword: "",
		},
		Embedded: true,
	}.New(nil)
	if err := frame.Connect(ctx); err != nil {
		return fmt.Errorf("connect frame: %w", err)
	}
	registerBuildEvents(frame)

	task := build.BuildTaskConfig{
		BuildTaskWorldConfig: build.BuildTaskWorldConfig{
			WorldPath:      sourceWorldPath,
			WorldStartPos:  sourceStartPos,
			WorldEndPos:    sourceEndPos,
			WorldDimension: sourceDimension,
		},
		BuildTaskBuildConfig: build.BuildTaskBuildConfig{
			ChunkGroupSide: &chunkGroupSide,
		},
		StartPos:  targetStartPos,
		Dimension: targetDimension,
		Speed:     &speed,
	}.NewTask(frame)

	frame.AddTask(task)
	if err := frame.Start(); err != nil {
		return fmt.Errorf("start build frame: %w", err)
	}
	return nil
}

func registerBuildEvents(frame define.Frame) {
	startedAt := time.Now()
	groupStartedAt := time.Now()

	frame.EventBus().Subscribe(build.EventNameInitStart, func() {
		startedAt = time.Now()
		fmt.Printf("[%s] init start\n", time.Now().Format(time.RFC3339))
	})
	frame.EventBus().Subscribe(build.EventNameInitOpenWorld, func(worldPath string) {
		fmt.Printf("[%s] init open world: %s\n", time.Now().Format(time.RFC3339), worldPath)
	})
	frame.EventBus().Subscribe(build.EventNameInitFinish, func() {
		fmt.Printf("[%s] init finish elapsed=%s\n", time.Now().Format(time.RFC3339), time.Since(startedAt))
	})

	frame.EventBus().Subscribe(build.EventNameRunStart, func(size define.Size, total int) {
		startedAt = time.Now()
		fmt.Printf("[%s] run start size=%dx%dx%d chunks=%d total_groups=%d\n",
			time.Now().Format(time.RFC3339), size.Width, size.Height, size.Length, size.ChunkCount(), total)
	})
	frame.EventBus().Subscribe(build.EventNameRunChunkGroupStart, func(progress int) {
		groupStartedAt = time.Now()
		fmt.Printf("[%s] chunk group start progress=%d\n", time.Now().Format(time.RFC3339), progress)
	})
	frame.EventBus().Subscribe(build.EventNameRunChunkGroupMove, func(groupPos define.ChunkPos, targetPos define.BlockPos) {
		fmt.Printf("[%s] chunk group move group=%v target=%v\n", time.Now().Format(time.RFC3339), groupPos, targetPos)
	})
	frame.EventBus().Subscribe(build.EventNameRunChunkGroupWaitLoadStart, func(groupPos define.ChunkPos) {
		fmt.Printf("[%s] wait load start group=%v\n", time.Now().Format(time.RFC3339), groupPos)
	})
	frame.EventBus().Subscribe(build.EventNameRunChunkGroupWaitLoadProbe, func(groupPos define.ChunkPos, attempt int, ready bool, timeout bool, message string) {
		fmt.Printf("[%s] wait load probe group=%v attempt=%d ready=%t timeout=%t message=%q\n",
			time.Now().Format(time.RFC3339), groupPos, attempt, ready, timeout, message)
	})
	frame.EventBus().Subscribe(build.EventNameRunChunkGroupWaitLoadRetry, func(groupPos define.ChunkPos, attempt int, timeout bool, message string) {
		fmt.Printf("[%s] wait load retry group=%v attempt=%d timeout=%t message=%q\n",
			time.Now().Format(time.RFC3339), groupPos, attempt, timeout, message)
	})
	frame.EventBus().Subscribe(build.EventNameRunChunkGroupWaitLoadFinish, func(groupPos define.ChunkPos) {
		fmt.Printf("[%s] wait load finish group=%v\n", time.Now().Format(time.RFC3339), groupPos)
	})
	frame.EventBus().Subscribe(build.EventNameRunChunkGroupLoaded, func(chunks map[define.ChunkPos]*chunk.Chunk, nbts map[define.ChunkPos][]map[string]any) {
		fmt.Printf("[%s] chunk group loaded chunks=%d nbts=%d\n", time.Now().Format(time.RFC3339), len(chunks), len(nbts))
	})
	frame.EventBus().Subscribe(build.EventNameRunCommandsGenerated, func(commandCount int) {
		fmt.Printf("[%s] commands generated count=%d\n", time.Now().Format(time.RFC3339), commandCount)
	})
	/*
		frame.EventBus().Subscribe(build.EventNameRunCommandSent, func(command string) {
			fmt.Printf("[%s] command sent %q\n", time.Now().Format(time.RFC3339), command)
		})
	*/
	frame.EventBus().Subscribe(build.EventNameRunChunkGroupFinish, func() {
		fmt.Printf("[%s] chunk group finish elapsed=%s\n", time.Now().Format(time.RFC3339), time.Since(groupStartedAt))
	})
	frame.EventBus().Subscribe(build.EventNameRunFinish, func() {
		fmt.Printf("[%s] run finish elapsed=%s\n", time.Now().Format(time.RFC3339), time.Since(startedAt))
	})
}
