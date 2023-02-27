/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package metric

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/sustainable-computing-io/kepler/pkg/bpfassets/attacher"
	"github.com/sustainable-computing-io/kepler/pkg/config"
	"github.com/sustainable-computing-io/kepler/pkg/power/accelerator"
	"k8s.io/klog/v2"
)

var (
	// ProcessMetricNames holds the list of names of the container metric
	ProcessMetricNames []string
	// ProcessFloatFeatureNames holds the feature name of the container float collector_metric. This is specific for the machine-learning based models.
	ProcessFloatFeatureNames []string = []string{}
	// ProcessUintFeaturesNames holds the feature name of the container utint collector_metric. This is specific for the machine-learning based models.
	ProcessUintFeaturesNames []string
	// ProcessFeaturesNames holds all the feature name of the container collector_metric. This is specific for the machine-learning based models.
	ProcessFeaturesNames []string
)

type ProcessMetrics struct {
	PID          uint64
	Command      string
	CounterStats map[string]*UInt64Stat
	// ebpf metrics
	CPUTime           *UInt64Stat
	SoftIRQCount      []UInt64Stat
	GPUStats          map[string]*UInt64Stat
	DynEnergyInCore   *UInt64Stat
	DynEnergyInDRAM   *UInt64Stat
	DynEnergyInUncore *UInt64Stat
	DynEnergyInPkg    *UInt64Stat
	DynEnergyInGPU    *UInt64Stat
	DynEnergyInOther  *UInt64Stat

	IdleEnergyInCore   *UInt64Stat
	IdleEnergyInDRAM   *UInt64Stat
	IdleEnergyInUncore *UInt64Stat
	IdleEnergyInPkg    *UInt64Stat
	IdleEnergyInGPU    *UInt64Stat
	IdleEnergyInOther  *UInt64Stat
}

// NewProcessMetrics creates a new ProcessMetrics instance
func NewProcessMetrics(pid uint64, command string) *ProcessMetrics {
	p := &ProcessMetrics{
		PID:                pid,
		Command:            command,
		CPUTime:            &UInt64Stat{},
		CounterStats:       make(map[string]*UInt64Stat),
		SoftIRQCount:       make([]UInt64Stat, config.MaxIRQ),
		DynEnergyInCore:    &UInt64Stat{},
		DynEnergyInDRAM:    &UInt64Stat{},
		DynEnergyInUncore:  &UInt64Stat{},
		DynEnergyInPkg:     &UInt64Stat{},
		DynEnergyInOther:   &UInt64Stat{},
		DynEnergyInGPU:     &UInt64Stat{},
		IdleEnergyInCore:   &UInt64Stat{},
		IdleEnergyInDRAM:   &UInt64Stat{},
		IdleEnergyInUncore: &UInt64Stat{},
		IdleEnergyInPkg:    &UInt64Stat{},
		IdleEnergyInOther:  &UInt64Stat{},
		IdleEnergyInGPU:    &UInt64Stat{},
	}

	for _, metricName := range AvailableHWCounters {
		p.CounterStats[metricName] = &UInt64Stat{}
	}
	// TODO: transparently list the other metrics and do not initialize them when they are not supported, e.g. HC
	if accelerator.IsGPUCollectionSupported() {
		p.CounterStats[config.GPUSMUtilization] = &UInt64Stat{}
		p.CounterStats[config.GPUMemUtilization] = &UInt64Stat{}
	}
	return p
}

// ResetCurr reset all current value to 0
func (p *ProcessMetrics) ResetDeltaValues() {
	p.CPUTime.ResetDeltaValues()
	for counterKey := range p.CounterStats {
		p.CounterStats[counterKey].ResetDeltaValues()
	}
	p.DynEnergyInCore.ResetDeltaValues()
	p.DynEnergyInDRAM.ResetDeltaValues()
	p.DynEnergyInUncore.ResetDeltaValues()
	p.DynEnergyInPkg.ResetDeltaValues()
	p.DynEnergyInOther.ResetDeltaValues()
	p.DynEnergyInGPU.ResetDeltaValues()
	p.IdleEnergyInCore.ResetDeltaValues()
	p.IdleEnergyInDRAM.ResetDeltaValues()
	p.IdleEnergyInUncore.ResetDeltaValues()
	p.IdleEnergyInPkg.ResetDeltaValues()
	p.IdleEnergyInOther.ResetDeltaValues()
	p.IdleEnergyInGPU.ResetDeltaValues()
}

// getFloatCurrAndAggrValue return curr, aggr float64 values of specific uint metric
func (p *ProcessMetrics) getFloatCurrAndAggrValue(metric string) (curr, aggr float64, err error) {
	// TO-ADD
	return 0, 0, nil
}

// getIntDeltaAndAggrValue return curr, aggr uint64 values of specific uint metric
func (p *ProcessMetrics) getIntDeltaAndAggrValue(metric string) (curr, aggr uint64, err error) {
	if val, exists := p.CounterStats[metric]; exists {
		return val.Delta, val.Aggr, nil
	}

	switch metric {
	// ebpf metrics
	case config.CPUTime:
		return p.CPUTime.Delta, p.CPUTime.Aggr, nil
	case config.IRQBlockLabel:
		return p.SoftIRQCount[attacher.IRQBlock].Delta, p.SoftIRQCount[attacher.IRQBlock].Aggr, nil
	case config.IRQNetTXLabel:
		return p.SoftIRQCount[attacher.IRQNetTX].Delta, p.SoftIRQCount[attacher.IRQNetTX].Aggr, nil
	case config.IRQNetRXLabel:
		return p.SoftIRQCount[attacher.IRQNetRX].Delta, p.SoftIRQCount[attacher.IRQNetRX].Aggr, nil
	}
	klog.V(4).Infof("cannot extract: %s", metric)
	return 0, 0, fmt.Errorf("cannot extract: %s", metric)
}

// ToEstimatorValues return values regarding metricNames
func (p *ProcessMetrics) ToEstimatorValues() (values []float64) {
	for _, metric := range ContainerFloatFeatureNames {
		curr, _, _ := p.getFloatCurrAndAggrValue(metric)
		values = append(values, curr)
	}
	for _, metric := range ContainerUintFeaturesNames {
		curr, _, _ := p.getIntDeltaAndAggrValue(metric)
		values = append(values, float64(curr))
	}
	return
}

// GetBasicValues return basic label balues
func (p *ProcessMetrics) GetBasicValues() []string {
	command := p.Command
	if len(command) > 10 {
		command = command[:10]
	}
	return []string{command}
}

// ToPrometheusValue return the value regarding metric label
func (p *ProcessMetrics) ToPrometheusValue(metric string) string {
	currentValue := false
	if strings.Contains(metric, "curr_") {
		currentValue = true
		metric = strings.ReplaceAll(metric, "curr_", "")
	}
	metric = strings.ReplaceAll(metric, "total_", "")

	if curr, aggr, err := p.getIntDeltaAndAggrValue(metric); err == nil {
		if currentValue {
			return strconv.FormatUint(curr, 10)
		}
		return strconv.FormatUint(aggr, 10)
	}
	if curr, aggr, err := p.getFloatCurrAndAggrValue(metric); err == nil {
		if currentValue {
			return fmt.Sprintf("%f", curr)
		}
		return fmt.Sprintf("%f", aggr)
	}
	klog.Errorf("cannot extract metric: %s", metric)
	return ""
}

func (p *ProcessMetrics) SumAllDynDeltaValues() uint64 {
	return p.DynEnergyInPkg.Delta + p.DynEnergyInGPU.Delta + p.DynEnergyInOther.Delta
}

func (p *ProcessMetrics) SumAllDynAggrValues() uint64 {
	return p.DynEnergyInPkg.Aggr + p.DynEnergyInGPU.Aggr + p.DynEnergyInOther.Aggr
}

func (p *ProcessMetrics) String() string {
	return fmt.Sprintf("energy from process pid: %d comm: %s\n"+
		"\tDyn ePkg (mJ): %s (eCore: %s eDram: %s eUncore: %s) eGPU (mJ): %s eOther (mJ): %s \n"+
		"\tIdle ePkg (mJ): %s (eCore: %s eDram: %s eUncore: %s) eGPU (mJ): %s eOther (mJ): %s \n"+
		"\tCPUTime:  %d (%d)\n"+
		"\tcounters: %v\n",
		p.PID, p.Command,
		p.DynEnergyInPkg, p.DynEnergyInCore, p.DynEnergyInDRAM, p.DynEnergyInUncore, p.DynEnergyInGPU, p.DynEnergyInOther,
		p.IdleEnergyInPkg, p.IdleEnergyInCore, p.IdleEnergyInDRAM, p.IdleEnergyInUncore, p.IdleEnergyInGPU, p.IdleEnergyInOther,
		p.CPUTime.Delta, p.CPUTime.Aggr,
		p.CounterStats)
}