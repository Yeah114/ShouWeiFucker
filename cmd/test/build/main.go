package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strconv"
	"strings"

	client "github.com/EmptyDea-Team/EmptyDea-core-client"
	"github.com/Yeah114/Fatalder/define"
	fatalder_frame "github.com/Yeah114/Fatalder/frame"
	"github.com/Yeah114/Fatalder/frame/task/build"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	var (
		coreAddr       = flag.String("core", "127.0.0.1:50051", "EmptyDeaCore gRPC address")
		worldPath      = flag.String("world", "", "source Bedrock world directory, .mcworld, or .zip")
		worldStartText = flag.String("world-start", "0,0,0", "source start block position: x,y,z")
		worldEndText   = flag.String("world-end", "15,0,15", "source end block position: x,y,z")
		startText      = flag.String("start", "0,0,0", "target start block position: x,y,z")
		worldDimension = flag.Int("world-dimension", int(define.DimensionIDOverworld), "source dimension id")
		dimension      = flag.Int("dimension", int(define.DimensionIDOverworld), "target dimension id")
		speed          = flag.Int("speed", 3000, "command send rate")
		chunkGroupSide = flag.Int("chunk-group-side", 2, "chunk group side length")
		noWait         = flag.Bool("no-wait", false, "disable chunk-load wait probe")
		noFill         = flag.Bool("no-fill", false, "disable fill command grouping")
	)
	flag.Parse()

	if strings.TrimSpace(*worldPath) == "" {
		return fmt.Errorf("missing -world")
	}

	worldStart, err := parseBlockPos(*worldStartText)
	if err != nil {
		return fmt.Errorf("parse -world-start: %w", err)
	}
	worldEnd, err := parseBlockPos(*worldEndText)
	if err != nil {
		return fmt.Errorf("parse -world-end: %w", err)
	}
	startPos, err := parseBlockPos(*startText)
	if err != nil {
		return fmt.Errorf("parse -start: %w", err)
	}

	ctx := context.Background()
	coreClient, conn, err := client.Dial(ctx, *coreAddr)
	if err != nil {
		return fmt.Errorf("dial core %q: %w", *coreAddr, err)
	}
	defer conn.Close()

	frame := fatalder_frame.New(coreClient)
	task := (&build.BuildTaskConfig{
		BuildTaskWorldConfig: build.BuildTaskWorldConfig{
			WorldPath:      *worldPath,
			WorldStartPos:  worldStart,
			WorldEndPos:    worldEnd,
			WorldDimension: define.Dimension(*worldDimension),
		},
		BuildTaskAutoConfig: build.BuildTaskAutoConfig{
			DisableAutoWaitChunkLoad: *noWait,
			DisableAutoFillBuildMode: *noFill,
		},
		BuildTaskBuildConfig: build.BuildTaskBuildConfig{
			ChunkGroupSide: chunkGroupSide,
		},
		StartPos:  startPos,
		Dimension: define.Dimension(*dimension),
		Speed:     speed,
	}).NewTask(frame)

	frame.AddTask(task)
	if err := frame.Start(); err != nil {
		return fmt.Errorf("start build frame: %w", err)
	}
	return nil
}

func parseBlockPos(text string) (define.BlockPos, error) {
	parts := strings.Split(text, ",")
	if len(parts) != 3 {
		return define.BlockPos{}, fmt.Errorf("expected x,y,z")
	}

	var pos define.BlockPos
	for i, part := range parts {
		value, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil {
			return define.BlockPos{}, fmt.Errorf("parse coordinate %d: %w", i, err)
		}
		pos[i] = value
	}
	return pos, nil
}
