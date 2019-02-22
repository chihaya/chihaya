package core

import (
	"github.com/ProtocolONE/chihaya/frontend/cord/config"
	"github.com/ProtocolONE/chihaya/frontend/cord/database"
	"go.uber.org/zap"
)

var inited = false

func InitCord() error {

	if inited {
		return nil
	}

	logger, _ := zap.NewProduction()
	zap.ReplaceGlobals(logger)
	defer logger.Sync()

	_, err := config.Init()
	if err != nil {
		return err
	}

	err = database.Init()
	if err != nil {
		return err
	}

	err = fillMemDB()
	if err != nil {
		return err
	}

	inited = true

	return nil
}

func fillMemDB() error {

	manager := database.NewTorrentManager()

	torrents, err := manager.FindAll()
	if err != nil {
		return err
	}

	memManager := database.NewMemTorrentManager()

	for _, t := range torrents {

		memManager.Insert(t)
	}

	return nil
}
