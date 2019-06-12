package hiera

import (
	"encoding/json"
	"io"

	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
	"github.com/lyraproj/pcore/utils"
	"gopkg.in/yaml.v3"
)

func Render(c px.Context, renderAs string, value px.Value, out io.Writer) {
	switch renderAs {
	case `yaml`, `json`:
		var v interface{}
		if !value.Equals(px.Undef, nil) {
			rf := c.Reflector().Reflect(value)
			if rf.IsValid() && rf.CanInterface() {
				v = rf.Interface()
			} else {
				v = value.String()
			}
		}
		var bs []byte
		var err error
		if renderAs == `yaml` {
			bs, err = yaml.Marshal(v)
		} else {
			bs, err = json.Marshal(v)
		}
		if err != nil {
			panic(err)
		}
		utils.WriteString(out, string(bs))
	case `binary`:
		bi := types.CoerceTo(c, `lookup value`, types.DefaultBinaryType(), value).(*types.Binary)
		_, err := out.Write(bi.Bytes())
		if err != nil {
			panic(err)
		}
	case `s`:
		utils.Fprintln(out, value)
	}
}
