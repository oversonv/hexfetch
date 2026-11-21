package modules

import "time"

type datetime struct{}

func init() { Register(datetime{}) }

func (datetime) Name() string { return "datetime" }

func (datetime) Run(ctx Context) ([]Entry, error) {
	return Single(ctx, "Date & Time", time.Now().Format("2006-01-02 15:04:05")), nil
}
