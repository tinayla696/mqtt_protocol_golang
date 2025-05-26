package develop

import (
	"os"
	"path/filepath"
	"runtime/pprof"
)

type Profile struct {
	cpuFile *os.File
	memFile *os.File
}

func ProfileInit(dir string) (*Profile, error) {
	// CPU Profiling
	cpuf, err := os.Create(filepath.Join(dir, "cpu.prof"))
	if err != nil {
		return nil, err
	}
	if err := pprof.StartCPUProfile(cpuf); err != nil {
		return nil, err
	}

	// Memory Profiling
	memf, err := os.Create(filepath.Join(dir, "mem.prof"))
	if err != nil {
		return nil, err
	}
	if err := pprof.WriteHeapProfile(memf); err != nil {
		return nil, err
	}
	return &Profile{
		cpuFile: cpuf,
		memFile: memf,
	}, nil
}

func (p *Profile) Stop() {
	p.cpuFile.Close()
	p.memFile.Close()
	pprof.StopCPUProfile()
}
