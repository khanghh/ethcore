package ethcore

import (
	"os"

	"github.com/ethereum/go-ethereum/log"
)

func InitDefaultLogger(verbosity int) log.Logger {
	logHanlder := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(true)))
	logHanlder.Verbosity(log.Lvl(verbosity))
	logger := log.Root()
	logger.SetHandler(logHanlder)
	return logger
}
