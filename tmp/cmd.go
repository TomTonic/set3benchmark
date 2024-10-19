package main

import (
	"fmt"
	"time"

	"github.com/alecthomas/kong"
)

var cli struct {
	Add struct {
		Loadfactor struct {
			From             uint32        `arg:"" help:"First set size to benchmark (inclusive)." short:"f"`
			To               uint32        `arg:"" help:"Last set size to benchmark (inclusive)." short:"t"`
			AddsPerRound     uint32        `help:"Number of Add(prng.Uint64()) instructions between two time measurements. Balance the value between memory consumption (cache size/speed) and timer precision of your runtime environment (e.g., Windows=100ns)." short:"r" default:"50000"`
			RuntimePerConfig time.Duration `help:"Benchmarking time per configuration (combination of initial set size and number of values to add)." short:"c" default:"1.5s"`
			RuntimePerAdd    time.Duration `help:"Expected runtime per single Add(prng.Uint64()) instruction. Used to calculate the necessary number of iterations to meet the runtime-per-config and to predcict the total runtime of the benchmark." short:"a" default:"8ns"`
			Pstep            *float64      `help:"Uses percentage value to increase the initial set size in benchmark configurations until the limit is reached (see relative-limit or absolute-limit). You can either specify a pstep or an istep. Default is an istep of size 1." short:"p" xor:"Pstep, Istep"`
			Istep            *uint32       `help:"Uses integer value to increase the initial set size in benchmark configurations until the limit is reached (see relative-limit or absolute-limit). You can either specify a pstep or an istep. Default is an istep of size 1." short:"i" xor:"Pstep, Istep"`
			RelativeLimit    *float64      `help:"Increase initial (pre-allocated) set sizes until at least x% headroom are reached. You can either specify a relative-limit or an absolute-limit. Default is a relative-limit of 100%." default:"100.0" xor:"RelativeLimit, AbsoluteLimit"`
			AbsoluteLimit    *uint32       `help:"Increase initial (pre-allocated) set sizes until at least an initial set size of x is reached. You can either specify a relative-limit or an absolute-limit. Default is a relative-limit of 100%." xor:"RelativeLimit, AbsoluteLimit"`
		} `cmd:"" help:"Perform a loadfactor test using different initial (pre-allocated) set sizes. This benchmark creates empty sets of a defined size x and then adds y random numers via Add(prng.Uint64()). Any combination of x and y is called 'configration'."`
	} `cmd:"" help:"Benchmark adding random uint64 to an empty set."`
}

func main() {
	ctx := kong.Parse(&cli,
		kong.Name("cmd"),
		kong.Description("A benchmark program to compare set implementations."),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
			Summary: true,
		}))

	switch ctx.Command() {
	case "add loadfactor <from> <to>":
		fmt.Println(cli.Add.Loadfactor.From, cli.Add.Loadfactor.To, cli.Add.Loadfactor.AddsPerRound, cli.Add.Loadfactor.RuntimePerConfig, cli.Add.Loadfactor.RuntimePerAdd)
		fmt.Println(cli.Add.Loadfactor.Pstep, cli.Add.Loadfactor.Istep, cli.Add.Loadfactor.RelativeLimit, cli.Add.Loadfactor.AbsoluteLimit)
	default:
		fmt.Print("Check ctx.Command(): ")
		fmt.Println(ctx.Command())
	}
}
