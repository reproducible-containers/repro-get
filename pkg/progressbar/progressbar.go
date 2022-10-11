/*
   Portions from https://github.com/lima-vm/lima/blob/v0.12.0/pkg/downloader/downloader.go
   Copyright (C) Lima authors.
   Licensed under the Apache License, Version 2.0
*/

package progressbar

import (
	"os"
	"time"

	"github.com/cheggaaa/pb/v3"
	"github.com/mattn/go-isatty"
)

// New is from https://github.com/lima-vm/lima/blob/v0.12.0/pkg/downloader/downloader.go#L291-L310
func New(size int64) (*pb.ProgressBar, error) {
	bar := pb.New64(size)

	bar.Set(pb.Bytes, true)
	if isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd()) {
		bar.SetTemplateString(`{{counters . }} {{bar . | green }} {{percent .}} {{speed . "%s/s"}}`)
		bar.SetRefreshRate(200 * time.Millisecond)
	} else {
		bar.Set(pb.Terminal, false)
		bar.Set(pb.ReturnSymbol, "\n")
		bar.SetTemplateString(`{{counters . }} ({{percent .}}) {{speed . "%s/s"}}`)
		bar.SetRefreshRate(5 * time.Second)
	}
	bar.SetWidth(80)
	if err := bar.Err(); err != nil {
		return nil, err
	}

	return bar, nil
}
