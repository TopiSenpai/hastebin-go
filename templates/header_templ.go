// Code generated by templ - DO NOT EDIT.

// templ: version: 0.2.476
package templates

//lint:file-ignore SA4006 This context is only used if a nested component is present.

import "github.com/a-h/templ"
import "context"
import "io"
import "bytes"

func header(vars DocumentVars) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, templ_7745c5c3_W io.Writer) (templ_7745c5c3_Err error) {
		templ_7745c5c3_Buffer, templ_7745c5c3_IsBuffer := templ_7745c5c3_W.(*bytes.Buffer)
		if !templ_7745c5c3_IsBuffer {
			templ_7745c5c3_Buffer = templ.GetBuffer()
			defer templ.ReleaseBuffer(templ_7745c5c3_Buffer)
		}
		ctx = templ.InitializeContext(ctx)
		templ_7745c5c3_Var1 := templ.GetChildren(ctx)
		if templ_7745c5c3_Var1 == nil {
			templ_7745c5c3_Var1 = templ.NopComponent
		}
		ctx = templ.ClearChildren(ctx)
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("<header><a title=\"gobin\" id=\"title\" href=\"/\">")
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		templ_7745c5c3_Var2 := `gobin`
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString(templ_7745c5c3_Var2)
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("</a> <a title=\"GitHub\" id=\"github\" class=\"icon-btn\" href=\"https://github.com/topi314/gobin\" target=\"_blank\"></a> <input id=\"nav-btn\" type=\"checkbox\"> <label title=\"Open Navigation\" class=\"hamb\" for=\"nav-btn\"><span></span></label><nav><a title=\"New\" id=\"new\" class=\"icon-btn\" href=\"/\" target=\"_blank\"></a> <button title=\"Save\" id=\"save\" class=\"icon-btn\"")
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		if !vars.Edit {
			_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString(" style=\"display: none;\"")
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
		}
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("></button> <button title=\"Edit\" id=\"edit\" class=\"icon-btn\"")
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		if vars.Edit {
			_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString(" style=\"display: none;\"")
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
		}
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("></button> <button title=\"Delete\" id=\"delete\" class=\"icon-btn\" disabled></button> <button title=\"Copy\" id=\"copy\" class=\"icon-btn\"></button> <button title=\"Raw\" id=\"raw\" class=\"icon-btn\"")
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		if !vars.Edit {
			_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString(" disabled")
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
		}
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("></button> <button title=\"Share\" id=\"share\" class=\"icon-btn\" disabled></button></nav></header>")
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		if !templ_7745c5c3_IsBuffer {
			_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteTo(templ_7745c5c3_W)
		}
		return templ_7745c5c3_Err
	})
}
