package config

import "github.com/Spelunkern/Shaiya-GX/internal/gameini"

type GameINI = gameini.File

func ReadGameINI(path string) (GameINI, error) { return gameini.Read(path) }
