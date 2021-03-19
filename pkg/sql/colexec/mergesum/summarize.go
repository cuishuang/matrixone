package mergesum

import (
	"matrixbase/pkg/container/batch"
	"matrixbase/pkg/encoding"
	"matrixbase/pkg/vm/process"
	"matrixbase/pkg/vm/register"
)

func Prepare(proc *process.Process, arg interface{}) error {
	n := arg.(*Argument)
	n.Attrs = make([]string, len(n.Es))
	for i, e := range n.Es {
		n.Attrs[i] = e.Alias
	}
	return nil
}

func Call(proc *process.Process, arg interface{}) (bool, error) {
	n := arg.(*Argument)
	for i := 0; i < len(proc.Reg.Ws); i++ {
		reg := proc.Reg.Ws[i]
		v := <-reg.Ch
		if v == nil {
			reg.Wg.Done()
			proc.Reg.Ws = append(proc.Reg.Ws[:i], proc.Reg.Ws[i:]...)
			i--
			continue
		}
		bat := v.(*batch.Batch)
		for _, e := range n.Es {
			vec, err := bat.GetVector(e.Name, proc)
			if err != nil {
				return false, err
			}
			if err := e.Agg.Fill(bat.Sels, vec); err != nil {
				return false, err
			}
		}
		reg.Wg.Done()
		bat.Clean(proc)
	}
	bat := batch.New(true, n.Attrs)
	{
		var err error
		for i, e := range n.Es {
			if bat.Vecs[i], err = e.Agg.Eval(proc); err != nil {
				bat.Vecs = bat.Vecs[:i]
				clean(bat, proc)
				return false, err
			}
			copy(bat.Vecs[i].Data, encoding.EncodeUint64(1+proc.Refer[n.Attrs[i]]))
		}
	}
	proc.Reg.Ax = bat
	register.FreeRegisters(proc)
	return false, nil
}

func clean(bat *batch.Batch, proc *process.Process) {
	bat.Clean(proc)
	register.FreeRegisters(proc)
}
