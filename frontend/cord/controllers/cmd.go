package controllers

import (
	"github.com/ProtocolONE/chihaya/frontend/cord/database"
	"github.com/ProtocolONE/chihaya/frontend/cord/models"

	"fmt"
	"github.com/labstack/echo"
	"go.uber.org/zap"
	"net/http"
)

// AddTorrent ...
func AddTorrent(context echo.Context) error {

	reqTorrent := &models.Torrent{}
	err := context.Bind(reqTorrent)
	if err != nil {
		return context.JSON(http.StatusBadRequest, models.Error{Code: models.ErrorInvalidJSONFormat, Message: "Invalid JSON format: " + err.Error()})
	}

	memManager := database.NewMemTorrentManager()
	memManager.Insert(reqTorrent)

	manager := database.NewTorrentManager()
	torrent, err := manager.FindByInfoHash(reqTorrent.InfoHash)
	if err != nil {
		return context.JSON(http.StatusBadRequest, models.Error{Code: models.ErrorReadDataBase, Message: fmt.Sprintf("Cannot read from database, error: %s", err.Error())})
	}

	if len(torrent) != 0 {
		return context.JSON(http.StatusBadRequest, models.Error{Code: models.ErrorTorrentAlreadyExists, Message: fmt.Sprintf("Torrent %s already exists", reqTorrent.InfoHash)})
	}

	err = manager.Insert(&models.Torrent{InfoHash: reqTorrent.InfoHash})
	if err != nil {
		return context.JSON(http.StatusBadRequest, models.Error{Code: models.ErrorAddTorrent, Message: fmt.Sprintf("Cannot add torrent %s, error: %s", reqTorrent.InfoHash, err.Error())})
	}

	zap.S().Infow("Added new torrent", zap.String("info_hash", reqTorrent.InfoHash))

	return context.NoContent(http.StatusOK)
}

// DeleteTorrent ...
func DeleteTorrent(context echo.Context) error {

	reqTorrent := &models.Torrent{}
	err := context.Bind(reqTorrent)
	if err != nil {
		return context.JSON(http.StatusBadRequest, models.Error{Code: models.ErrorInvalidJSONFormat, Message: "Invalid JSON format: " + err.Error()})
	}

	memManager := database.NewMemTorrentManager()
	memManager.RemoveByInfoHash(reqTorrent.InfoHash)

	manager := database.NewTorrentManager()
	err = manager.RemoveByInfoHash(reqTorrent.InfoHash)
	if err != nil {
		return context.JSON(http.StatusBadRequest, models.Error{Code: models.ErrorDeleteTorrent, Message: fmt.Sprintf("Cannot delete torrent %s, error: %s", reqTorrent.InfoHash, err.Error())})
	}

	zap.S().Infow("Removed torrent", zap.String("info_hash", reqTorrent.InfoHash))

	return context.NoContent(http.StatusOK)
}
