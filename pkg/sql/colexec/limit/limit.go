package limit

import (
	"bytes"
	"fmt"
	"matrixbase/pkg/container/batch"
	"matrixbase/pkg/encoding"
	"matrixbase/pkg/vm/mempool"
	"matrixbase/pkg/vm/process"
	"matrixbase/pkg/vm/register"
)

func String(arg interface{}, buf *bytes.Buffer) {
	n := arg.(*Argument)
	buf.WriteString(fmt.Sprintf("limit(%v)", n.Limit))
}

func Prepare(_ *process.Process, _ interface{}) error {
	return nil
}

func Call(proc *process.Process, arg interface{}) (bool, error) {
	if proc.Reg.Ax == nil {
		return false, nil
	}
	n := arg.(*Argument)
	bat := proc.Reg.Ax.(*batch.Batch)
	if length := uint64(len(bat.Sels)); length > 0 {
		newSeen := n.Seen + length
		if newSeen >= n.Limit { // limit - seen
			bat.Sels = bat.Sels[:n.Limit-n.Seen]
			proc.Reg.Ax = bat
			register.FreeRegisters(proc)
			return true, nil
		}
		n.Seen = newSeen
		proc.Reg.Ax = bat
		register.FreeRegisters(proc)
		return false, nil
	}
	length, err := bat.Length(proc)
	if err != nil {
		clean(bat, proc)
		return false, err
	}
	newSeen := n.Seen + uint64(length)
	if newSeen >= n.Limit { // limit - seen
		data, sels, err := newSels(int64(n.Limit-n.Seen), proc)
		if err != nil {
			clean(bat, proc)
			return true, err
		}
		bat.Sels = sels
		bat.SelsData = data
		proc.Reg.Ax = bat
		register.FreeRegisters(proc)
		return true, nil
	}
	n.Seen = newSeen
	proc.Reg.Ax = bat
	register.FreeRegisters(proc)
	return false, nil
}

func newSels(count int64, proc *process.Process) ([]byte, []int64, error) {
	data, err := proc.Alloc(count * 8)
	if err != nil {
		return nil, nil, err
	}
	sels := encoding.DecodeInt64Slice(data[mempool.CountSize:])
	for i := int64(0); i < count; i++ {
		sels[i] = i
	}
	return data, sels[:count], nil
}

func clean(bat *batch.Batch, proc *process.Process) {
	bat.Clean(proc)
	register.FreeRegisters(proc)
}
