package main

import (
	"fmt"
	"log"

	"github.com/EmptyDea-Team/bedrock-world-operator/block"
	"github.com/EmptyDea-Team/bedrock-world-operator/chunk"
	"github.com/EmptyDea-Team/bedrock-world-operator/define"
	"github.com/EmptyDea-Team/bedrock-world-operator/world"
)

const (
	testWorldPath = "/sd/sdcard/bwo"

	testBlockX = uint8(1)
	testBlockY = int16(64)
	testBlockZ = uint8(1)
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	dimension := define.Dimension(define.DimensionIDOverworld)
	chunkPos := define.ChunkPos{0, 0}
	table := block.NewBlockRuntimeIDTable(true)

	db, err := world.Open(testWorldPath, nil, table)
	if err != nil {
		return fmt.Errorf("open world: %w", err)
	}

	c, exists, err := db.LoadChunk(dimension, chunkPos)
	if err != nil {
		_ = db.Close()
		return fmt.Errorf("load chunk %v: %w", chunkPos, err)
	}
	if !exists {
		c = chunk.NewChunk(table.AirRuntimeID(), dimension.Range())
	}

	diamondRuntimeID, found := table.StateToRuntimeID("minecraft:diamond_block", map[string]any{})
	if !found {
		_ = db.Close()
		return fmt.Errorf("find runtime id for minecraft:diamond_block")
	}

	c.SetBlock(testBlockX, testBlockY, testBlockZ, 0, diamondRuntimeID)
	if err := db.SaveChunk(dimension, chunkPos, c); err != nil {
		_ = db.Close()
		return fmt.Errorf("save chunk %v: %w", chunkPos, err)
	}
	if err := db.CloseWorld(); err != nil {
		return fmt.Errorf("close world after save: %w", err)
	}

	reopened, err := world.Open(testWorldPath, nil, table)
	if err != nil {
		return fmt.Errorf("reopen world: %w", err)
	}
	defer reopened.CloseWorld()

	savedChunk, exists, err := reopened.LoadChunk(dimension, chunkPos)
	if err != nil {
		return fmt.Errorf("reload chunk %v: %w", chunkPos, err)
	}
	if !exists {
		return fmt.Errorf("reload chunk %v: chunk does not exist", chunkPos)
	}

	gotRuntimeID := savedChunk.Block(testBlockX, testBlockY, testBlockZ, 0)
	gotName, gotProperties, found := table.RuntimeIDToState(gotRuntimeID)
	if !found {
		return fmt.Errorf("read back block runtime id %d: state not found", gotRuntimeID)
	}
	if gotRuntimeID != diamondRuntimeID {
		return fmt.Errorf("read back block: got %s%v (%d), want minecraft:diamond_block (%d)", gotName, gotProperties, gotRuntimeID, diamondRuntimeID)
	}

	fmt.Printf("ok: opened %q, wrote %s%v at chunk %v local (%d,%d,%d), saved and verified\n",
		testWorldPath,
		gotName,
		gotProperties,
		chunkPos,
		testBlockX,
		testBlockY,
		testBlockZ,
	)
	return nil
}
