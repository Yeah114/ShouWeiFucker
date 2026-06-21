package main

import (
	"context"
	"fmt"
	"log"

	client "github.com/EmptyDea-Team/EmptyDea-core-client"
	"github.com/Yeah114/Fatalder/define"
	"github.com/Yeah114/Fatalder/frame"
	"github.com/Yeah114/Fatalder/frame/task/build"
)

const (
	coreAddr = "127.0.0.1:50051"

	sourceWorldPath = "/path/to/source.mcworld"

	sourceDimension = define.Dimension(define.DimensionIDOverworld)
	targetDimension = define.Dimension(define.DimensionIDOverworld)

	speed          = 3000
	chunkGroupSide = 2
)

var (
	sourceStartPos = define.BlockPos{0, 0, 0}
	sourceEndPos   = define.BlockPos{15, 0, 15}
	targetStartPos = define.BlockPos{0, 0, 0}
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx := context.Background()
	coreClient, conn, err := client.Dial(ctx, coreAddr)
	if err != nil {
		return fmt.Errorf("dial core %q: %w", coreAddr, err)
	}
	defer conn.Close()

	frame := frame.New(coreClient)
	task := build.BuildTaskConfig{
		BuildTaskWorldConfig: build.BuildTaskWorldConfig{
			WorldPath:      sourceWorldPath,
			WorldStartPos:  sourceStartPos,
			WorldEndPos:    sourceEndPos,
			WorldDimension: sourceDimension,
		},
		BuildTaskBuildConfig: build.BuildTaskBuildConfig{
			ChunkGroupSide: intPtr(chunkGroupSide),
		},
		StartPos:  targetStartPos,
		Dimension: targetDimension,
		Speed:     intPtr(speed),
	}.NewTask(frame)

	frame.AddTask(task)
	if err := frame.Start(); err != nil {
		return fmt.Errorf("start build frame: %w", err)
	}
	return nil
}

func intPtr(value int) *int {
	return &value
}
