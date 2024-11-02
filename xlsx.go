package main

import (
	"fmt"
	"runtime"
	"strings"

	misc "github.com/TomTonic/set3benchmark/misc"
	cpuid "github.com/klauspost/cpuid/v2"
	"github.com/xuri/excelize/v2"
)

type ExcelOutput struct {
	excelFile *excelize.File
	FileName  string
	cursor    map[string]int
}

func NewExcelOutput(fileName string) *ExcelOutput {
	f := excelize.NewFile()
	eo := &ExcelOutput{
		excelFile: f,
		FileName:  fileName,
		cursor:    make(map[string]int),
	}
	return eo
}

func (eo *ExcelOutput) Save() {
	if err := eo.excelFile.SaveAs(eo.FileName); err != nil {
		fmt.Println(err)
	}
}

func (eo *ExcelOutput) Close() {
	if err := eo.excelFile.SaveAs(eo.FileName); err != nil {
		fmt.Println(err)
	}
	if err := eo.excelFile.Close(); err != nil {
		fmt.Println(err)
	}
}

func (eo *ExcelOutput) WriteLine(sheetName string, startCol int, values ...interface{}) {
	row := eo.getNextRow(sheetName)
	for i, v := range values {
		cellName, _ := excelize.CoordinatesToCellName(startCol+i, row)
		eo.excelFile.SetCellValue(sheetName, cellName, v)
	}
}

func (eo *ExcelOutput) getNextRow(sheetName string) int {
	row, found := eo.cursor[sheetName]
	if !found {
		_, err := eo.excelFile.NewSheet(sheetName)
		if err != nil {
			fmt.Println(err)
		}
		eo.cursor[sheetName] = 2
		return 1
	}
	eo.cursor[sheetName] = row + 1
	return row
}

func (eo *ExcelOutput) WriteConfigSheet(p benchmarkSetup) {
	cfgSheetName := "Summary"
	index, err := eo.excelFile.NewSheet(cfgSheetName)
	if err != nil {
		fmt.Println(err)
		return
	}
	eo.WriteLine(cfgSheetName, 1, "Architecture (GOARCH)", runtime.GOARCH)
	eo.WriteLine(cfgSheetName, 1, "OS (GOOS)", runtime.GOOS)
	eo.WriteLine(cfgSheetName, 1, "CPU Name", cpuid.CPU.BrandName)
	eo.WriteLine(cfgSheetName, 1, "CPU Vendor ID", cpuid.CPU.VendorID)
	eo.WriteLine(cfgSheetName, 1, "CPU Vendor String (raw)", cpuid.CPU.VendorString)
	eo.WriteLine(cfgSheetName, 1, "CPU Family", cpuid.CPU.Family)
	eo.WriteLine(cfgSheetName, 1, "CPU Model", cpuid.CPU.Model)
	eo.WriteLine(cfgSheetName, 1, "CPU Stepping", cpuid.CPU.Stepping)
	eo.WriteLine(cfgSheetName, 1, "CPU PhysicalCores", cpuid.CPU.PhysicalCores)
	eo.WriteLine(cfgSheetName, 1, "CPU ThreadsPerCore", cpuid.CPU.ThreadsPerCore)
	eo.WriteLine(cfgSheetName, 1, "CPU LogicalCores", cpuid.CPU.LogicalCores)
	eo.WriteLine(cfgSheetName, 1, "CPU Cacheline", cpuid.CPU.CacheLine, "bytes")
	eo.WriteLine(cfgSheetName, 1, "CPU L1 Data Cache:", cpuid.CPU.Cache.L1D, "bytes")
	eo.WriteLine(cfgSheetName, 1, "CPU L1 Instruction Cache:", cpuid.CPU.Cache.L1I, "bytes")
	eo.WriteLine(cfgSheetName, 1, "CPU L2 Cache:", cpuid.CPU.Cache.L2, "bytes")
	eo.WriteLine(cfgSheetName, 1, "CPU L3 Cache:", cpuid.CPU.Cache.L3, "bytes")
	eo.WriteLine(cfgSheetName, 1, "CPU Frequency", cpuid.CPU.Hz, "Hz", "(0 means unknown)")
	eo.WriteLine(cfgSheetName, 1, "CPU Boost Frequency", cpuid.CPU.BoostFreq, "Hz", "(0 means unknown)")
	eo.WriteLine(cfgSheetName, 1, "CPU Features:", strings.Join(cpuid.CPU.FeatureSet(), ", "))

	eo.WriteLine(cfgSheetName, 1, "")
	eo.WriteLine(cfgSheetName, 1, "Actual runtime per SampleTime()-call", misc.GetSampleTimeRuntime(), "ns/call")
	eo.WriteLine(cfgSheetName, 1, "Maximum timer precision", misc.GetSampleTimePrecision(), "ns")
	overhead, qerror := getPRNGOverhead()
	eo.WriteLine(cfgSheetName, 1, "Actual runtime per prng.Uint64()-call", overhead, "ns/call", "actual quantization error", qerror*overhead, "ns/call")
	eo.WriteLine(cfgSheetName, 1, "Expected runtime per Add(prng.Uint64())-call", p.expRuntimePerAdd, "ns/call")
	quantizationError := calcQuantizationError(p)
	eo.WriteLine(cfgSheetName, 1, "Number of Add(prng.Uint64())-calls per round", p.targetAddsPerRound, "calls/round", "expected quantization error", quantizationError*p.expRuntimePerAdd, "ns/call")
	eo.WriteLine(cfgSheetName, 1, "Rounds per configuration", p.totalAddsPerConfig/p.targetAddsPerRound, "rounds")
	eo.WriteLine(cfgSheetName, 1, "Number of Add(prng.Uint64())-calls per configuration", p.totalAddsPerConfig, "calls")
	eo.WriteLine(cfgSheetName, 1, "Expected runtime per configuration", p.secondsPerConfig, "sec.")

	eo.WriteLine(cfgSheetName, 1, "")
	numberOfConfigs := getNumberOfConfigs(p.fromSetSize, p.toSetSize, p.Pstep, p.Istep, p.RelativeLimit, p.AbsoluteLimit)
	eo.WriteLine(cfgSheetName, 1, "Number of configurations", numberOfConfigs)
	eo.WriteLine(cfgSheetName, 1, "Set size from", p.fromSetSize, "elements")
	eo.WriteLine(cfgSheetName, 1, "Set size to", p.toSetSize, "elements")
	totalduration := predictTotalDuration(p)
	eo.WriteLine(cfgSheetName, 1, "Expected total runtime", totalduration)

	// Set active sheet of the workbook.
	eo.excelFile.SetActiveSheet(index)
	if err := eo.excelFile.SaveAs(eo.FileName); err != nil {
		fmt.Println(err)
	}
}
